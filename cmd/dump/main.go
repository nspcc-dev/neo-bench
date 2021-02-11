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
	"github.com/nspcc-dev/neo-go/pkg/network/payload"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"go.uber.org/zap"
)

// wifTo is a wif of the wallet where all NEO and GAS are sent.
const wifTo = "KxhEDBQyyEFymvfJD96q8stMbJMbZUb6D1PmXqBWZDU2WvbvVs9o"

// txPerBlock is a new policy value for maximum amount of transactions per block
// which is set to be uint16 max value - 1.
const txPerBlock = 65534

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

func newNEP5Transfer(sc util.Uint160, from, to util.Uint160, amount int64) *transaction.Transaction {
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter, sc, "transfer", callflag.All, from, to, amount, nil)
	emit.Opcodes(w.BinWriter, opcode.ASSERT)

	script := w.Bytes()
	tx := transaction.New(netmode.PrivNet, script, 11000000)
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

	// update max tx per block
	var policyHash, _ = util.Uint160DecodeStringLE("79bcd398505eb779df6e67e4be6c14cded08e2f2")
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter, policyHash, "setMaxTransactionsPerBlock", callflag.All, int64(txPerBlock))
	emit.AppCall(w.BinWriter, policyHash, "setMaxBlockSize", callflag.All, int64(payload.MaxSize/2))
	script := w.Bytes()
	txUpdatePolicy := transaction.New(netmode.PrivNet, script, 10000000)
	if *isSingle {
		txUpdatePolicy.NetworkFee = 1500000
	} else {
		txUpdatePolicy.NetworkFee = 4600000
	}
	txUpdatePolicy.ValidUntilBlock = 1000
	txUpdatePolicy.Signers = append(txUpdatePolicy.Signers, transaction.Signer{
		Account: c.addr,
		Scopes:  transaction.CalledByEntry,
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
