package internal

import (
	"context"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
)

const (
	conflictNetworkFee      int64  = 2_000_000
	conflictValidUntilBlock uint32 = 1200
)

func newConflictTxSetBuilder(opts BenchOptions) txSetBuilder {
	if len(opts.Senders) == 0 {
		log.Fatal("conflict scenario: no senders available")
	}

	return func(idx int) ([]*transaction.Transaction, *wallet.Account) {
		sender := opts.Senders[idx%len(opts.Senders)]
		txA, txB := newConflictingPair(idx, sender)
		return []*transaction.Transaction{txA, txB}, wallet.NewAccountFromPrivateKey(sender)
	}
}

func newConflictingPair(idx int, sender *keys.PrivateKey) (txA, txB *transaction.Transaction) {
	txA = newTx(sender, uint32(2*idx))

	hashA := txA.Hash()

	txB = newTx(sender, uint32(2*idx+1))
	txB.Attributes = append(txB.Attributes, transaction.Attribute{
		Type:  transaction.ConflictsT,
		Value: &transaction.Conflicts{Hash: hashA},
	})

	return txA, txB
}

func newTx(sender *keys.PrivateKey, nonce uint32) *transaction.Transaction {
	tx := transaction.New([]byte{byte(opcode.RET)}, 1_000_000)
	tx.Nonce = nonce
	tx.NetworkFee = conflictNetworkFee
	tx.ValidUntilBlock = conflictValidUntilBlock
	tx.Signers = []transaction.Signer{{
		Account: sender.GetScriptHash(),
		Scopes:  transaction.None,
	}}
	return tx
}

func pickRandomNodePair(n int) (int, int) {
	a := rand.IntN(n)
	b := rand.IntN(n)
	for b == a {
		b = rand.IntN(n)
	}
	return a, b
}

func (d *doer) SendConflictPairs(ctx context.Context) {
	defer close(d.sentOut)

	nodeCount := d.cli.AddrCount()
	start := time.Now()

	for range d.wrkCount {
		d.waiter.Go(func() {
			done := ctx.Done()
			timer := time.NewTimer(d.timeLimit)
			defer timer.Stop()

			for {
				select {
				case <-done:
					return
				case <-timer.C:
					return
				default:
				}

				blobA, blobB, ok := d.claimConflictPair()
				if !ok {
					return
				}

				nodeA, nodeB := pickRandomNodePair(nodeCount)

				var wgPair sync.WaitGroup
				wgPair.Add(2)
				go func() {
					defer wgPair.Done()
					d.handleSendResult(blobA, d.cli.SendTXToNode(ctx, blobA, nodeA), start)
				}()
				go func() {
					defer wgPair.Done()
					d.handleSendResult(blobB, d.cli.SendTXToNode(ctx, blobB, nodeB), start)
				}()
				wgPair.Wait()
			}
		})
	}
	d.waiter.Wait()

	d.reportSendResult(start)
}

func (d *doer) claimConflictPair() (string, string, bool) {
	d.Lock()
	defer d.Unlock()

	if d.dump.TransactionsQueue.Len() < 2 {
		return "", "", false
	}

	a, err := d.dump.TransactionsQueue.Get()
	if err != nil {
		return "", "", false
	}
	b, err := d.dump.TransactionsQueue.Get()
	if err != nil {
		return "", "", false
	}

	return a.(string), b.(string), true
}
