package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core/native"
	"github.com/nspcc-dev/neo-go/pkg/core/native/nativenames"
	"github.com/nspcc-dev/neo-go/pkg/core/state"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/rpc/client"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/manifest"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/nef"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
)

// wifTo is a wif of the wallet where all NEO and GAS are sent.
const wifTo = "KxhEDBQyyEFymvfJD96q8stMbJMbZUb6D1PmXqBWZDU2WvbvVs9o"

// Prepare sends prepare transactions on chain at runtime.
func (d *doer) Prepare(ctx context.Context) {
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

	err = fillChain(ctx, c, sgn)
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

func fillChain(ctx context.Context, c *client.Client, sgn *signer) error {
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

	priv, err := keys.NewPrivateKeyFromWIF(wifTo)
	if err != nil {
		return err
	}

	txMoveNeo := newNEP5Transfer(isSingle, neoHash, sgn.addr, priv.GetScriptHash(), native.NEOTotalSupply)
	txMoveGas := newNEP5Transfer(isSingle, gasHash, sgn.addr, priv.GetScriptHash(), native.GASFactor*2900000)
	sgn.signTx(txMoveNeo, txMoveGas)

	log.Println("Sending NEO and GAS transfer tx")
	err = sendTx(ctx, c, txMoveNeo, txMoveGas)
	if err != nil {
		return err
	}

	err = awaitTx(ctx, timeout,
		func() (bool, error) {
			b, err := c.NEP17BalanceOf(neoHash, priv.GetScriptHash())
			log.Println("NEO balance:", b)
			return b > 0, err
		},
		func() (bool, error) {
			b, err := c.NEP17BalanceOf(gasHash, priv.GetScriptHash())
			log.Println("GAS balance:", b)
			return b > 0, err
		})
	if err != nil {
		return err
	}

	// We deploy contract from priv to avoid having different hashes for single/4-node benchmarks.
	// The contract is taken from `examples/token` of neo-go with 2 minor corrections:
	// 1. Owner address is replaced with the address of WIF we use.
	// 2. All funds are minted to owner in `_deploy`.
	txDeploy, h, err := newDeployTx(mgmtHash, priv, "/tokencontract/token.nef",
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
