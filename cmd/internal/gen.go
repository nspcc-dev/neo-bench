package internal

import (
	"context"
	"encoding/base64"
	"encoding/binary"
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
		BenchOptions      BenchOptions
		TransactionsQueue *queue.RingBuffer
	}

	// GenerateCallback used to do something with hash and marshaled transactions when generates.
	GenerateCallback func(hash, blob string) error

	txBlob struct {
		hash, blob string
	}

	txRequest struct {
		tx  *transaction.Transaction
		acc *wallet.Account
	}

	txSetBuilder func(idx int) (txs []*transaction.Transaction, acc *wallet.Account)
)

const (
	// NEOTransfer is the type of NEO transfer tx.
	NEOTransfer = "neo"
	// GASTransfer is the type of GAS transfer tx.
	GASTransfer = "gas"
	// ContractTransfer is the type of deployed NEP17 contract transfer tx.
	ContractTransfer = "nep17"
	// ConflictTransfer is the type of conflicting transaction pairs.
	ConflictTransfer = "conflict"
)

// newNEOTransferTx returns NEO transfer transaction with random nonce.
func newNEOTransferTx(p *keys.PrivateKey, to util.Uint160) *transaction.Transaction {
	neoContractHash, _ := util.Uint160DecodeBytesBE([]byte(neo.Hash))
	return newTransferTx(p, neoContractHash, to)
}

// newGASTransferTx returns GAS transfer transaction with random nonce.
func newGASTransferTx(p *keys.PrivateKey, to util.Uint160) *transaction.Transaction {
	gasContractHash, _ := util.Uint160DecodeBytesBE([]byte(gas.Hash))
	return newTransferTx(p, gasContractHash, to)
}

func newTransferTx(p *keys.PrivateKey, contractHash, toAddr util.Uint160) *transaction.Transaction {
	fromAddressHash := p.GetScriptHash()
	w := io.NewBufBinWriter()
	emit.AppCall(w.BinWriter,
		contractHash, "transfer", callflag.All,
		fromAddressHash, toAddr, int64(1), nil)
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

// Generate runs opts.TxCount tasks in parallel, one task per worker
// iteration: a single transfer transaction per task for the standard
// transfer types, or a conflicting pair per task for ConflictTransfer.
func Generate(ctx context.Context, opts BenchOptions, callback ...GenerateCallback) *Dump {
	start := time.Now()
	taskCount := int(opts.TxCount)
	build, perTask := newTxSetBuilder(opts)

	dump := Dump{
		TransactionsQueue: queue.NewRingBuffer(opts.TxCount * uint64(perTask)),
	}

	if perTask == 1 {
		log.Printf("Generate %d txs", taskCount)
	} else {
		log.Printf("Generate %d conflicting transaction pairs (%d txs)", taskCount, taskCount*perTask)
	}

	reqCh := make([]chan int, genWorkerCount)
	for i := range reqCh {
		reqCh[i] = make(chan int, 1)
	}
	resultCh := make([]chan []txBlob, genWorkerCount)
	for i := range resultCh {
		resultCh[i] = make(chan []txBlob, 1)
	}

	var wg sync.WaitGroup
	for i := range genWorkerCount {
		wg.Go(func() {
			genWorker(build, reqCh[i], resultCh[i])
		})
	}

	finishCh := make(chan struct{})
	go func() {
		for i := range taskCount {
			if ctx.Err() != nil {
				log.Fatal(ctx.Err())
			}

			reqCh[i%len(reqCh)] <- i
		}
		for _, ch := range reqCh {
			close(ch)
		}
		close(finishCh)
	}()

	for i := range taskCount {
		blobs := <-resultCh[i%len(resultCh)]

		for _, r := range blobs {
			if err := dump.TransactionsQueue.Put(r.blob); err != nil {
				log.Fatalf("Cannot enqueue transaction #%d: %s", i, err)
			}
			for j := range callback {
				if err := callback[j](r.hash, r.blob); err != nil {
					log.Fatalf("Callback returns error: %d %v", i, err)
				}
			}
		}
	}

	<-finishCh
	wg.Wait()
	for _, ch := range resultCh {
		close(ch)
	}

	log.Printf("Done: %s", time.Since(start))
	return &dump
}

func newTxSetBuilder(opts BenchOptions) (txSetBuilder, int) {
	if strings.ToLower(opts.TransferType) == ConflictTransfer {
		return newConflictTxSetBuilder(opts), 2
	}
	return newTransferTxSetBuilder(opts), 1
}

func newTransferTxSetBuilder(opts BenchOptions) txSetBuilder {
	// We support both N-to-1 and 1-to-N cases, thus the size is adjusted.
	txR := make([]txRequest, max(len(opts.Senders), opts.ToCount))
	for i := range txR {
		sender := opts.Senders[i%len(opts.Senders)]
		receiver := opts.Senders[0].GetScriptHash()
		if opts.ToCount > 1 {
			rem := (i + 1) % opts.ToCount
			if rem < len(opts.Senders) {
				receiver = opts.Senders[rem].GetScriptHash()
			} else { // support up to 65536 receivers
				receiver = util.Uint160{}
				binary.LittleEndian.PutUint16(receiver[:], uint16(rem))
			}
		}

		var tx *transaction.Transaction
		switch strings.ToLower(opts.TransferType) {
		case NEOTransfer:
			tx = newNEOTransferTx(sender, receiver)
		case GASTransfer:
			tx = newGASTransferTx(sender, receiver)
		case ContractTransfer:
			h, _ := util.Uint160DecodeStringLE("ceb508fc02abc2dc27228e21976699047bbbcce0")
			tx = newTransferTx(sender, h, receiver)
		default:
			panic(fmt.Sprintf("invalid type: %s", opts.TransferType))
		}
		txR[i].tx = tx
		txR[i].acc = wallet.NewAccountFromPrivateKey(sender)
	}

	return func(idx int) ([]*transaction.Transaction, *wallet.Account) {
		r := txR[idx%len(txR)]
		tx := *r.tx
		tx.Nonce = uint32(idx)
		return []*transaction.Transaction{&tx}, r.acc
	}
}

func genWorker(build txSetBuilder, ch <-chan int, out chan<- []txBlob) {
	buf := io.NewBufBinWriter()
	for idx := range ch {
		txs, acc := build(idx)

		blobs := make([]txBlob, len(txs))
		for i, tx := range txs {
			if err := acc.SignTx(netmode.PrivNet, tx); err != nil {
				log.Fatalf("Could not sign tx: %v", err)
			}

			buf.Reset()
			tx.EncodeBinary(buf.BinWriter)
			if buf.Err != nil {
				log.Fatalf("Could not prepare transaction: %d %v", idx, buf.Err)
			}

			blobs[i] = txBlob{
				hash: tx.Hash().String(),
				blob: base64.StdEncoding.EncodeToString(buf.Bytes()),
			}
		}

		out <- blobs
	}
}
