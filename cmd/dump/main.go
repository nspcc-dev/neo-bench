package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/config"
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core"
	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/native"
	"github.com/nspcc-dev/neo-go/pkg/core/storage"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/rpc/client"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"go.uber.org/zap"
)

// wifTo is a wif of the wallet where all NEO and GAS are sent.
const wifTo = "KxhEDBQyyEFymvfJD96q8stMbJMbZUb6D1PmXqBWZDU2WvbvVs9o"

// txPerBlock is a new policy value for maximum amount of transactions per block.
const txPerBlock = 40000

var (
	isSingle = flag.Bool("single", false, "generate dump for a single node")
	outName = flag.String("out", "dump.acc", "file where to write dump")
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
		Base: block.Base{
			Network:       netmode.PrivNet,
			PrevHash:      hdr.Hash(),
			Timestamp:     uint64(time.Now().UTC().Unix())*1000 + uint64(index),
			Index:         index,
			NextConsensus: c.addr,
		},
		ConsensusData: block.ConsensusData{
			PrimaryIndex: 0,
			Nonce:        1111,
		},
		Transactions: txs,
	}

	err = b.RebuildMerkleRoot()
	if err != nil {
		return err
	}

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

func newNEP5Transfer(sc util.Uint160, from, to util.Uint160, amount int64) *transaction.Transaction {
	w := io.NewBufBinWriter()
	emit.AppCallWithOperationAndArgs(w.BinWriter, sc, "transfer", from, to, amount)
	emit.Opcode(w.BinWriter, opcode.ASSERT)

	script := w.Bytes()
	tx := transaction.New(netmode.PrivNet, script, 0)
	tx.NetworkFee = 274000
	tx.ValidUntilBlock = 1000
	tx.Sender = from
	tx.Cosigners = append(tx.Cosigners, transaction.Cosigner{
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

	txMoveNeo := newNEP5Transfer(client.NeoContractHash, c.addr, priv.GetScriptHash(), native.NEOTotalSupply)
	txMoveGas := newNEP5Transfer(client.GasContractHash, c.addr, priv.GetScriptHash(), native.GASFactor*29000000)
	c.signTx(txMoveNeo, txMoveGas)

	err = addBlock(bc, c, txMoveNeo, txMoveGas)
	if err != nil {
		return err
	}

	// update max tx per block
	w := io.NewBufBinWriter()
	emit.AppCallWithOperationAndArgs(w.BinWriter, client.PolicyContractHash, "setMaxTransactionsPerBlock", int64(txPerBlock))
	emit.Opcode(w.BinWriter, opcode.ASSERT)
	script := w.Bytes()
	txUpdatePolicy := transaction.New(netmode.PrivNet, script, 0)
	txUpdatePolicy.NetworkFee = 250000
	txUpdatePolicy.ValidUntilBlock = 1000
	txUpdatePolicy.Sender = c.addr
	txUpdatePolicy.Cosigners = append(txUpdatePolicy.Cosigners, transaction.Cosigner{
		Account:          c.addr,
		Scopes:           transaction.CalledByEntry,
	})
	c.signTx(txUpdatePolicy)

	err = addBlock(bc, c, txUpdatePolicy)
	if err != nil {
		return err
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
