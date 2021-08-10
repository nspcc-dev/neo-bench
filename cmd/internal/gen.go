package internal

import (
	"context"
	"encoding/base64"
	"log"
	"time"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/neo"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
)

type (
	// Dump contains hashes and marshaled transactions.
	Dump struct {
		TransactionsQueue *queue.RingBuffer
	}

	// GenerateCallback used to do something with hash and marshaled transactions when generates.
	GenerateCallback func(hash, blob string) error
)

// getWif returns Wif.
func getWif() (*keys.WIF, error) {
	var (
		wifEncoded = "KxhEDBQyyEFymvfJD96q8stMbJMbZUb6D1PmXqBWZDU2WvbvVs9o"
		version    = byte(0x00)
	)
	return keys.WIFDecode(wifEncoded, version)
}

// newTX returns Invocation transaction with some random attributes in order to have different hashes.
func newTX(wif *keys.WIF) *transaction.Transaction {
	fromAddressHash := wif.PrivateKey.GetScriptHash()
	neoContractHash, _ := util.Uint160DecodeBytesBE([]byte(neo.Hash))

	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter,
		neoContractHash, "transfer", callflag.All,
		fromAddressHash, fromAddressHash, int64(1), nil)
	emit.Opcodes(w.BinWriter, opcode.ASSERT)
	if w.Err != nil {
		panic(w.Err)
	}

	script := w.Bytes()
	tx := transaction.New(script, 10000000)
	tx.NetworkFee = 1500000 // hardcoded for now
	tx.ValidUntilBlock = 1200
	tx.Signers = append(tx.Signers, transaction.Signer{
		Account: fromAddressHash,
		Scopes:  transaction.CalledByEntry,
	})
	return tx
}

// Generate used to generate the specified number of transactions.
func Generate(ctx context.Context, count int, callback ...GenerateCallback) *Dump {
	start := time.Now()

	dump := Dump{
		TransactionsQueue: queue.NewRingBuffer(uint64(count)),
	}

	log.Printf("Generate %d txs", count)

	wif, err := getWif()
	if err != nil {
		log.Fatalf("Could not get wif: %v", err)
	}

	acc, err := wallet.NewAccountFromWIF(wif.S)
	if err != nil {
		log.Fatalf("Could not create account: %v", err)
	}

	buf := io.NewBufBinWriter()

	tx := newTX(wif)
	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			log.Fatal(ctx.Err())
		}

		tx := *tx
		tx.Nonce = uint32(i)

		if err := acc.SignTx(netmode.PrivNet, &tx); err != nil {
			log.Fatalf("Could not sign tx: %v", err)
		}

		tx.EncodeBinary(buf.BinWriter)

		if buf.Err != nil {
			log.Fatalf("Could not prepare transaction: %d %v", i, err)
		}

		hash := tx.Hash().String()
		blob := base64.StdEncoding.EncodeToString(buf.Bytes())

		err := dump.TransactionsQueue.Put(blob)
		if err != nil {
			log.Fatalf("Cannot enqueue transaction #%d: %s", i, err)
		}

		for j := range callback {
			if err := callback[j](hash, blob); err != nil {
				log.Fatalf("Callback returns error: %d %v", i, err)
			}
		}

		buf.Reset()
	}

	log.Printf("Done: %s", time.Since(start))
	return &dump
}
