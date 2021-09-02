package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core/native"
	"github.com/nspcc-dev/neo-go/pkg/core/native/nativenames"
	"github.com/nspcc-dev/neo-go/pkg/core/state"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/rpc/client"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/manifest"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/nef"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/vm/stackitem"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
)

// Prepare sends prepare transactions on chain at runtime.
func (d *doer) Prepare(ctx context.Context, vote bool, opts BenchOptions) {
	log.Println("Prepare chain for benchmark")

	// Preparation stage isn't done during main benchmark,
	// so using native client doesn't play a big role.
	c, err := client.New(ctx, d.cli.addr[0], client.Options{})
	if err != nil {
		log.Fatalf("could not create client: %v", err)
	}

	err = c.Init()
	if err != nil {
		log.Fatalf("could not init client: %v", err)
	}

	isSingle, err := isSingleNode(c)
	if err != nil {
		log.Fatalf("could not get the number of validators: %v", err)
	}

	log.Printf("Determined single node setup: %t", isSingle)

	sgn, err := initChain(isSingle)
	if err != nil {
		log.Fatalf("could not initialize chain: %v", err)
	}

	err = fillChain(ctx, c, sgn, vote, opts)
	if err != nil {
		log.Fatalf("could not create blocks: %v", err)
	}
}

func isSingleNode(c *client.Client) (bool, error) {
	// Committee can be bigger than consensus nodes, but it is not the case in our setup.
	// Querying `GetNextBlockValidators` can return empty list.
	vs, err := c.GetCommittee()
	if err != nil {
		return false, err
	}

	if len(vs) == 0 {
		return false, errors.New("received empty committee")
	}
	return len(vs) == 1, nil
}

func initChain(single bool) (*signer, error) {
	var wifs []string
	if single {
		wifs = []string{"KxyjQ8eUa4FHt3Gvioyt1Wz29cTUrE4eTqX3yFSk1YFCsPL8uNsY"}
	} else {
		wifs = []string{
			"KzfPUYDC9n2yf4fK5ro4C8KMcdeXtFuEnStycbZgX3GomiUsvX6W",
			"KzgWE3u3EDp13XPXXuTKZxeJ3Gi8Bsm8f9ijY3ZsCKKRvZUo1Cdn",
			"KxyjQ8eUa4FHt3Gvioyt1Wz29cTUrE4eTqX3yFSk1YFCsPL8uNsY",
			"L2oEXKRAAMiPEZukwR5ho2S6SMeQLhcK9mF71ZnF7GvT8dU4Kkgz",
		}
	}

	return newSigner(wifs...)
}

func newDeployTx(mgmtHash util.Uint160, priv *keys.PrivateKey, nefName, manifestName string) (*transaction.Transaction, util.Uint160, error) {
	rawNef, err := ioutil.ReadFile(nefName)
	if err != nil {
		return nil, util.Uint160{}, err
	}

	rawManifest, err := ioutil.ReadFile(manifestName)
	if err != nil {
		return nil, util.Uint160{}, err
	}

	buf := io.NewBufBinWriter()
	emit.AppCall(buf.BinWriter, mgmtHash, "deploy", callflag.All, rawNef, rawManifest)
	if buf.Err != nil {
		return nil, util.Uint160{}, buf.Err
	}

	tx := transaction.New(buf.Bytes(), 100*native.GASFactor)
	tx.Signers = []transaction.Signer{{Account: priv.GetScriptHash(), Scopes: transaction.Global}}
	tx.ValidUntilBlock = 1000
	tx.NetworkFee = 10_000000

	// Contract hash is immutable so we calculate it once and then reuse during tx generation.
	ne, err := nef.FileFromBytes(rawNef)
	if err != nil {
		return nil, util.Uint160{}, err
	}
	m := new(manifest.Manifest)
	if err := json.Unmarshal(rawManifest, m); err != nil {
		return nil, util.Uint160{}, err
	}
	h := state.CreateContractHash(tx.Sender(), ne.Checksum, m.Name)
	log.Printf("Contract hash: %s\n", h.StringLE())

	acc := wallet.NewAccountFromPrivateKey(priv)
	return tx, h, acc.SignTx(netmode.PrivNet, tx)
}

func newNEP5Transfer(isSingle bool, sc util.Uint160, from, to util.Uint160, amount int64) *transaction.Transaction {
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter, sc, "transfer", callflag.All, from, to, amount, nil)
	emit.Opcodes(w.BinWriter, opcode.ASSERT)

	script := w.Bytes()
	tx := transaction.New(script, 11000000)
	if isSingle {
		tx.NetworkFee = 1500000
	} else {
		tx.NetworkFee = 4500000
	}
	tx.ValidUntilBlock = 1000
	tx.Signers = append(tx.Signers, transaction.Signer{
		Account: from,
		Scopes:  transaction.CalledByEntry,
	})
	return tx
}

func fillChain(ctx context.Context, c *client.Client, sgn *signer, vote bool, opts BenchOptions) error {
	cs, err := c.GetNativeContracts()
	if err != nil {
		return err
	}

	isSingle := len(sgn.privs) == 1

	// timeout is block time x3
	timeout := time.Second * 15
	if isSingle {
		timeout = time.Second * 3
	}

	var neoHash, gasHash, mgmtHash util.Uint160
	for i := range cs {
		switch cs[i].Manifest.Name {
		case nativenames.Neo:
			neoHash = cs[i].Hash
		case nativenames.Gas:
			gasHash = cs[i].Hash
		case nativenames.Management:
			mgmtHash = cs[i].Hash
		}
	}

	if vote {
		err = registerCandidates(ctx, neoHash, c, sgn)
		if err != nil {
			return err
		}
	}

	txs := make([]*transaction.Transaction, 0, len(opts.Senders)*2)
	neoAmount := int64(native.NEOTotalSupply / len(opts.Senders))
	gasAmount := int64(native.GASFactor * 2900000 / len(opts.Senders))
	for _, priv := range opts.Senders {
		txMoveNeo := newNEP5Transfer(isSingle, neoHash, sgn.addr, priv.GetScriptHash(), neoAmount)
		txMoveGas := newNEP5Transfer(isSingle, gasHash, sgn.addr, priv.GetScriptHash(), gasAmount)
		sgn.signTx(txMoveNeo, txMoveGas)
		txs = append(txs, txMoveNeo, txMoveGas)
	}

	log.Println("Sending NEO and GAS transfer tx")
	err = sendTx(ctx, c, txs...)
	if err != nil {
		return err
	}

	fs := make([]func() (bool, error), 0, len(opts.Senders)*2)
	for i := 0; i < len(opts.Senders); i++ {
		addr := opts.Senders[i].GetScriptHash()
		fs = append(fs,
			func() (bool, error) {
				b, err := c.NEP17BalanceOf(neoHash, addr)
				return b > 0, err
			},
			func() (bool, error) {
				b, err := c.NEP17BalanceOf(gasHash, addr)
				return b > 0, err
			})
	}
	err = awaitTx(ctx, timeout, fs...)

	if err != nil {
		return err
	}

	if vote {
		err = voteForCandidates(ctx, neoHash, c, sgn, opts.Senders)
		if err != nil {
			return err
		}
	}

	// We deploy contract from priv to avoid having different hashes for single/4-node benchmarks.
	// The contract is taken from `examples/token` of neo-go with 2 minor corrections:
	// 1. Owner address is replaced with the address of WIF we use.
	// 2. All funds are minted to owner in `_deploy`.
	txDeploy, h, err := newDeployTx(mgmtHash, opts.Senders[0], "/tokencontract/token.nef",
		"/tokencontract/token.manifest.json")
	if err != nil {
		return err
	}

	log.Println("Sending contract deploy tx")
	err = sendTx(ctx, c, txDeploy)
	if err != nil {
		return err
	}

	return awaitTx(ctx, timeout,
		func() (bool, error) {
			_, err := c.GetContractStateByHash(h)
			log.Println("Contract was persisted:", err == nil)
			return err == nil, err
		})
}

func registerCandidates(ctx context.Context, neoHash util.Uint160, c *client.Client, sgn *signer) error {
	for _, p := range sgn.privs {
		tx := newRegisterTx(neoHash, p, sgn)
		err := sendTx(ctx, c, tx)
		if err != nil {
			return err
		}
	}

	return awaitTx(ctx, time.Second*15, func() (bool, error) {
		res, err := c.InvokeFunction(neoHash, "getCandidates", []smartcontract.Parameter{}, nil)
		if err != nil {
			return false, err
		}
		return len(res.Stack) == 1 && len(res.Stack[0].Value().([]stackitem.Item)) == len(sgn.privs), nil
	})
}

func voteForCandidates(ctx context.Context, neoHash util.Uint160, c *client.Client, sgn *signer, senders []*keys.PrivateKey) error {
	for i := range senders {
		tx := newVoteTx(neoHash, senders[i], sgn.privs[i%len(sgn.privs)].PublicKey())
		err := sendTx(ctx, c, tx)
		if err != nil {
			return err
		}
	}

	return awaitTx(ctx, time.Second*15, func() (bool, error) {
		res, err := c.InvokeFunction(neoHash, "getCandidates", []smartcontract.Parameter{}, nil)
		if err != nil {
			return false, err
		}
		var cnt big.Int
		for _, it := range res.Stack[0].Value().([]stackitem.Item) {
			votes, err := it.Value().([]stackitem.Item)[1].TryInteger()
			if err != nil {
				return false, err
			}
			cnt.Add(&cnt, votes)
		}
		expected := int64(native.NEOTotalSupply / len(senders) * len(senders))
		return cnt.Int64() == expected, nil
	})
}
func newVoteTx(neoHash util.Uint160, priv *keys.PrivateKey, voteFor *keys.PublicKey) *transaction.Transaction {
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter,
		neoHash, "vote", callflag.All,
		priv.GetScriptHash(), voteFor.Bytes())
	emit.Opcodes(w.BinWriter, opcode.ASSERT)
	if w.Err != nil {
		panic(w.Err)
	}

	script := w.Bytes()
	tx := transaction.New(script, 15_000_000)
	tx.NetworkFee = 2000_000
	tx.ValidUntilBlock = 1200
	tx.Signers = append(tx.Signers, transaction.Signer{
		Account: priv.GetScriptHash(),
		Scopes:  transaction.CalledByEntry,
	})

	err := wallet.NewAccountFromPrivateKey(priv).SignTx(netmode.PrivNet, tx)
	if err != nil {
		panic(err)
	}
	return tx
}

func newRegisterTx(neoHash util.Uint160, priv *keys.PrivateKey, sgn *signer) *transaction.Transaction {
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter, neoHash, "registerCandidate",
		callflag.All, priv.PublicKey().Bytes())
	if w.Err != nil {
		panic(w.Err)
	}

	script := w.Bytes()
	tx := transaction.New(script, native.DefaultRegisterPrice+5_000_000)
	if len(sgn.privs) == 1 {
		tx.NetworkFee = 3000000
	} else {
		tx.NetworkFee = 6000000
	}
	tx.ValidUntilBlock = 1000
	tx.Signers = []transaction.Signer{
		{
			Account: sgn.addr,
			Scopes:  transaction.Global,
		},
		{
			Account: priv.GetScriptHash(),
			Scopes:  transaction.CalledByEntry,
		},
	}

	sgn.signTx(tx)
	err := wallet.NewAccountFromPrivateKey(priv).SignTx(netmode.PrivNet, tx)
	if err != nil {
		panic(err)
	}
	return tx
}

func sendTx(ctx context.Context, c *client.Client, txs ...*transaction.Transaction) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	for _, tx := range txs {
		_, err := c.SendRawTransaction(tx)
		if err != nil {
			return fmt.Errorf("could not send prepare tx: %w", err)
		}
	}
	return nil
}

func awaitTx(ctx context.Context, duration time.Duration, check ...func() (bool, error)) error {
	const retryInterval = time.Millisecond * 500

	attempts := duration / retryInterval

	for i := 0; i < len(check); {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ok, err := check[i]()
		if err != nil || !ok {
			if attempts == 0 {
				return errors.New("timeout while waiting for prepare tx to persist")
			}
			attempts--
			time.Sleep(retryInterval)
			continue
		}

		i++
	}
	return nil
}
