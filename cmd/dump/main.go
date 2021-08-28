package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core"
	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/blockchainer"
	"github.com/nspcc-dev/neo-go/pkg/core/native"
	"github.com/nspcc-dev/neo-go/pkg/core/state"
	"github.com/nspcc-dev/neo-go/pkg/core/storage"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/manifest"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/nef"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/trigger"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
	"go.uber.org/zap"
)

// wifTo is a wif of the wallet where all NEO and GAS are sent.
const wifTo = "KxhEDBQyyEFymvfJD96q8stMbJMbZUb6D1PmXqBWZDU2WvbvVs9o"

var (
	isSingle = flag.Bool("single", false, "generate dump for a single node")
	outName  = flag.String("out", "dump.acc", "file where to write dump")
)

func main() {
	flag.Parse()

	bc, c, err := initChain(*isSingle)
	if err != nil {
		log.Fatalf("could not initialize chain: %v", err)
	}
	defer bc.Close()

	err = fillChain(bc, c)
	if err != nil {
		log.Fatalf("could not create blocks: %v", err)
	}

	err = dumpChain(bc, *outName)
	if err != nil {
		log.Fatalf("could not dump chain: %v", err)
	}
}

func addBlock(bc *core.Blockchain, c *signer, txs ...*transaction.Transaction) error {
	height := int(bc.BlockHeight())
	h := bc.GetHeaderHash(height)
	hdr, err := bc.GetHeader(h)
	if err != nil {
		return err
	}

	index := uint32(height + 1)
	b := &block.Block{
		Header: block.Header{
			PrevHash:      hdr.Hash(),
			Timestamp:     uint64(time.Now().UTC().Unix())*1000 + uint64(index),
			Index:         index,
			NextConsensus: c.addr,
			PrimaryIndex:  0,
		},
		Transactions: txs,
	}

	b.RebuildMerkleRoot()

	c.signBlock(b)
	return bc.AddBlock(b)
}

func initChain(single bool) (*core.Blockchain, *signer, error) {
	const base = "../.docker/ir/"
	cfgPath := base + "go.protocol.privnet.one.yml"
	if single {
		cfgPath = base + "go.protocol.privnet.single.yml"
	}
	cfg, err := config.LoadFile(cfgPath)
	if err != nil {
		return nil, nil, err
	}

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

	c, err := newSigner(wifs...)
	if err != nil {
		return nil, nil, err
	}

	lg, err := zap.NewDevelopment()
	if err != nil {
		return nil, nil, err
	}
	bc, err := core.NewBlockchain(storage.NewMemoryStore(), cfg.ProtocolConfiguration, lg)
	if err != nil {
		return nil, nil, err
	}

	go bc.Run()
	return bc, c, nil
}

func newDeployTx(bc blockchainer.Blockchainer, priv *keys.PrivateKey, nefName, manifestName string) (*transaction.Transaction, error) {
	rawNef, err := ioutil.ReadFile(nefName)
	if err != nil {
		return nil, err
	}

	rawManifest, err := ioutil.ReadFile(manifestName)
	if err != nil {
		return nil, err
	}

	buf := io.NewBufBinWriter()
	emit.AppCall(buf.BinWriter, bc.ManagementContractHash(), "deploy", callflag.All, rawNef, rawManifest)
	if buf.Err != nil {
		return nil, buf.Err
	}

	tx := transaction.New(buf.Bytes(), 100*native.GASFactor)
	tx.Signers = []transaction.Signer{{Account: priv.GetScriptHash(), Scopes: transaction.Global}}
	tx.ValidUntilBlock = 1000
	tx.NetworkFee = 10_000000

	// Contract hash is immutable so we calculate it once and then reuse during tx generation.
	ne, err := nef.FileFromBytes(rawNef)
	if err != nil {
		return nil, err
	}
	m := new(manifest.Manifest)
	if err := json.Unmarshal(rawManifest, m); err != nil {
		return nil, err
	}
	h := state.CreateContractHash(tx.Sender(), ne.Checksum, m.Name)
	log.Printf("Contract hash: %s\n", h.StringLE())

	acc := wallet.NewAccountFromPrivateKey(priv)
	return tx, acc.SignTx(netmode.PrivNet, tx)
}

func newNEP5Transfer(sc util.Uint160, from, to util.Uint160, amount int64) *transaction.Transaction {
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter, sc, "transfer", callflag.All, from, to, amount, nil)
	emit.Opcodes(w.BinWriter, opcode.ASSERT)

	script := w.Bytes()
	tx := transaction.New(script, 11000000)
	if *isSingle {
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

func fillChain(bc *core.Blockchain, c *signer) error {
	priv, err := keys.NewPrivateKeyFromWIF(wifTo)
	if err != nil {
		return err
	}

	txMoveNeo := newNEP5Transfer(bc.GoverningTokenHash(), c.addr, priv.GetScriptHash(), native.NEOTotalSupply)
	txMoveGas := newNEP5Transfer(bc.UtilityTokenHash(), c.addr, priv.GetScriptHash(), native.GASFactor*29000000)
	c.signTx(txMoveNeo, txMoveGas)

	err = addBlock(bc, c, txMoveNeo, txMoveGas)
	if err != nil {
		return err
	}

	// We deploy contract from priv to avoid having different hashes for single/4-node benchmarks.
	// The contract is taken from `examples/token` of neo-go with 2 minor corrections:
	// 1. Owner address is replaced with the address of WIF we use.
	// 2. All funds are minted to owner in `_deploy`.
	txDeploy, err := newDeployTx(bc, priv, "./dump/tokencontract/token.nef",
		"./dump/tokencontract/token.manifest.json")
	if err != nil {
		return err
	}
	err = addBlock(bc, c, txDeploy)
	if err != nil {
		return err
	}

	st, _ := bc.GetAppExecResults(txMoveGas.Hash(), trigger.Application)
	if st[0].VMState != vm.HaltState {
		return errors.New(st[0].FaultException)
	}
	return addBlock(bc, c)
}

func dumpChain(bc *core.Blockchain, name string) error {
	outStream, err := os.Create(name)
	if err != nil {
		return err
	}
	defer outStream.Close()

	writer := io.NewBinWriterFromIO(outStream)

	count := bc.BlockHeight() + 1
	writer.WriteU32LE(count)

	for i := 0; i < int(count); i++ {
		bh := bc.GetHeaderHash(i)
		b, err := bc.GetBlock(bh)
		if err != nil {
			return err
		}
		w := io.NewBufBinWriter()
		b.EncodeBinary(w.BinWriter)
		if w.Err != nil {
			return w.Err
		}
		bytes := w.Bytes()
		writer.WriteU32LE(uint32(len(bytes)))
		writer.WriteBytes(bytes)
	}
	return writer.Err
}
