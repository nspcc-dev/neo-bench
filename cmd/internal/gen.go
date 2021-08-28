package internal

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/interop/native/gas"
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

	txBlob struct {
		hash, blob string
	}
)

const (
	// NEOTransfer is the type of NEO transfer tx.
	NEOTransfer = "neo"
	// GASTransfer is the type of GAS transfer tx.
	GASTransfer = "gas"
	// ContractTransfer is the type of deployed NEP17 contract transfer tx.
	ContractTransfer = "nep17"
)

var workerCount = runtime.NumCPU()

// getWif returns Wif.
func getWif() (*keys.WIF, error) {
	var (
		wifEncoded = "KxhEDBQyyEFymvfJD96q8stMbJMbZUb6D1PmXqBWZDU2WvbvVs9o"
		version    = byte(0x00)
	)
	return keys.WIFDecode(wifEncoded, version)
}

// newNEOTransferTx returns NEO transfer transaction with random nonce.
func newNEOTransferTx(wif *keys.WIF) *transaction.Transaction {
	neoContractHash, _ := util.Uint160DecodeBytesBE([]byte(neo.Hash))
	return newTransferTx(wif, neoContractHash)
}

// newGASTransferTx returns GAS transfer transaction with random nonce.
func newGASTransferTx(wif *keys.WIF) *transaction.Transaction {
	gasContractHash, _ := util.Uint160DecodeBytesBE([]byte(gas.Hash))
	return newTransferTx(wif, gasContractHash)
}

func newTransferTx(wif *keys.WIF, contractHash util.Uint160) *transaction.Transaction {
	fromAddressHash := wif.PrivateKey.GetScriptHash()

	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter,
		contractHash, "transfer", callflag.All,
		fromAddressHash, fromAddressHash, int64(1), nil)
	emit.Opcodes(w.BinWriter, opcode.ASSERT)
	if w.Err != nil {
		panic(w.Err)
	}

	script := w.Bytes()
	tx := transaction.New(script, 15000000)
	tx.NetworkFee = 1500000 // hardcoded for now
	tx.ValidUntilBlock = 1200
	tx.Signers = append(tx.Signers, transaction.Signer{
		Account: fromAddressHash,
		Scopes:  transaction.CalledByEntry,
	})
	return tx
}

var genWorkerCount = runtime.NumCPU()

// Generate used to generate the specified number of transactions.
func Generate(ctx context.Context, typ string, count int, callback ...GenerateCallback) *Dump {
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

	txCh := make(chan *transaction.Transaction, genWorkerCount)
	result := make(chan txBlob, genWorkerCount)

	var wg sync.WaitGroup
	wg.Add(genWorkerCount)
	for i := 0; i < genWorkerCount; i++ {
		go func(i int) {
			defer wg.Done()
			genTxWorker(i, acc, txCh, result)
		}(i)
	}

	var tx *transaction.Transaction
	switch strings.ToLower(typ) {
	case NEOTransfer:
		tx = newNEOTransferTx(wif)
	case GASTransfer:
		tx = newGASTransferTx(wif)
	case ContractTransfer:
		h, _ := util.Uint160DecodeStringLE("ceb508fc02abc2dc27228e21976699047bbbcce0")
		tx = newTransferTx(wif, h)
	default:
		panic(fmt.Sprintf("invalid type: %s", typ))
	}

	finishCh := make(chan struct{})
	go func() {
		for i := 0; i < count; i++ {
			if ctx.Err() != nil {
				log.Fatal(ctx.Err())
			}

			txCh <- tx
		}
		close(txCh)
		close(finishCh)
	}()

	for i := 0; i < count; i++ {
		r := <-result

		err := dump.TransactionsQueue.Put(r.blob)
		if err != nil {
			log.Fatalf("Cannot enqueue transaction #%d: %s", i, err)
		}

		for j := range callback {
			if err := callback[j](r.hash, r.blob); err != nil {
				log.Fatalf("Callback returns error: %d %v", i, err)
			}
		}
	}

	<-finishCh
	wg.Wait()
	close(result)

	log.Printf("Done: %s", time.Since(start))
	return &dump
}

func genTxWorker(n int, acc *wallet.Account, ch <-chan *transaction.Transaction, out chan<- txBlob) {
	baseNonce := n << 24 // 255 possible workers and 16M transactions should be enough
	i := 0

	buf := io.NewBufBinWriter()
	for tx := range ch {
		tx := *tx
		tx.Nonce = uint32(baseNonce | i)

		if err := acc.SignTx(netmode.PrivNet, &tx); err != nil {
			log.Fatalf("Could not sign tx: %v", err)
		}

		buf.Reset()
		tx.EncodeBinary(buf.BinWriter)

		if buf.Err != nil {
			log.Fatalf("Could not prepare transaction: %d %v", i, buf.Err)
		}

		out <- txBlob{
			hash: tx.Hash().String(),
			blob: base64.StdEncoding.EncodeToString(buf.Bytes()),
		}

		i++
	}
}
