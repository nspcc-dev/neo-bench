package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/neorpc/result"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient"
	"github.com/nspcc-dev/neo-go/pkg/rpcclient/actor"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/opcode"
	"github.com/nspcc-dev/neo-go/pkg/vm/vmstate"
	"github.com/nspcc-dev/neo-go/pkg/wallet"
)

const (
	// hasConflictsErrText is the text of HasConflicts verification result error.
	hasConflictsErrText = "HasConflicts"
	// hasConflictsErrText = "has conflicts" // Go-node version
	// invalidAttributeErrTest is the text of an error that is returned when transaction attribute verification fails.
	invalidAttributeErrTest = "InvalidAttribute"
	// invalidAttributeErrTest = "invalid attribute" // Go-node version
)

func main() {
	// Create RPC client.
	address := "localhost:20331" // address is an RPC node address
	c, err := rpcclient.New(context.Background(), "http://"+address, rpcclient.Options{})
	checkNoErr(err)

	// Check that connection to RPC node is OK and block count can be retrieved.
	count, err := c.GetBlockCount()
	checkNoErr(err)
	fmt.Printf("Block count: %d\n", count)

	// Retrieve accounts that owns all NEO and GAS (these are the signers of transactions).
	// One account is good, and another one will try to prevent the good one from accepting
	// its transactions.
	w, err := wallet.NewWalletFromFile("./my_wallet.json")
	checkNoErr(err)
	acc := w.Accounts[0]
	checkNoErr(acc.Decrypt("qwerty", w.Scrypt))
	maliciousAcc := w.Accounts[1]
	checkNoErr(maliciousAcc.Decrypt("qwerty", w.Scrypt))
	signer := actor.SignerAccount{
		Signer: transaction.Signer{
			Account: acc.ScriptHash(),
			Scopes:  transaction.Global,
		},
		Account: acc,
	}
	maliciousSigner := actor.SignerAccount{
		Signer: transaction.Signer{
			Account: maliciousAcc.ScriptHash(),
			Scopes:  transaction.Global,
		},
		Account: maliciousAcc,
	}
	fmt.Printf("Good signer: %s\nMalicious signer: %s\n", signer.Account.Address, maliciousSigner.Account.Address)

	// Create RPC actor that will generate all the transactions: one actor for good signer
	// and another actor for bad signer.
	act, err := actor.New(c, []actor.SignerAccount{signer})
	checkNoErr(err)
	maliciousAct, err := actor.New(c, []actor.SignerAccount{maliciousSigner})
	checkNoErr(err)

	// Generate a simple script for all transactions.
	script := []byte{byte(opcode.PUSH1)}

	// Define basic network fee that should be enough to pay for a simple transaction
	// with three conflict attributes. This fee will be used as smallNetworkFee.
	testTx, err := act.MakeTunedRun(script, []transaction.Attribute{
		{
			Type: transaction.ConflictsT,
			Value: &transaction.Conflicts{
				Hash: util.Uint256{1, 2, 3},
			},
		},
		{
			Type: transaction.ConflictsT,
			Value: &transaction.Conflicts{
				Hash: util.Uint256{1, 2, 4},
			},
		},
		{
			Type: transaction.ConflictsT,
			Value: &transaction.Conflicts{
				Hash: util.Uint256{1, 2, 5},
			},
		},
	}, actor.DefaultCheckerModifier)
	checkNoErr(err)
	smallNetFee := testTx.NetworkFee
	if smallNetFee < 1000_0000 {
		smallNetFee = 1000_0000 // rounded value is good.
	}
	fmt.Printf("`smallNetworkFee` value: %d GAS\n", smallNetFee)

	// Define function that generates new transaction (it doesn't send the transaction to the network, just generates).
	// `netFee` is a network fee that will be set for the transaction.
	// `act` is an actor that will be used to generate the transaction (transaction's signers depends on it).
	// `conflictingHashes` is the set of hashes that should be added to the transaction's Conflicts attributes (this set may be empty, then transaction won't have Conflicts attributes).
	getTx := func(netFee int64, act *actor.Actor, conflictingHashes ...util.Uint256) *transaction.Transaction {
		attrs := make([]transaction.Attribute, len(conflictingHashes))
		for i, h := range conflictingHashes {
			attrs[i] = transaction.Attribute{
				Type: transaction.ConflictsT,
				Value: &transaction.Conflicts{
					Hash: h,
				},
			}
		}
		tx, err := act.MakeTunedRun(script, attrs, func(r *result.Invoke, t *transaction.Transaction) error {
			err := actor.DefaultCheckerModifier(r, t)
			if err != nil {
				return err
			}
			t.NetworkFee = netFee
			return nil
		})
		checkNoErr(err)
		return tx
	}

	// Define two functions that will generate transactions with good/malicious sender and
	// specified conflicting hashes.
	getConflictsTx := func(netFee int64, hashes ...util.Uint256) *transaction.Transaction {
		return getTx(netFee, act, hashes...)
	}
	getMaliciousTx := func(netFee int64, hashes ...util.Uint256) *transaction.Transaction {
		return getTx(netFee, maliciousAct, hashes...)
	}

	// Wait for a new block to be accepted to start pooling transactions to a fresh mempool.
	// Block time is set to be 1 minute (quite large to fit the test's purposes).
	waitForNewBlock(c)

	// Start the test.
	fmt.Printf("\nStarting the mempool test.\n\n")

	// tx1 in mempool and does not conflict with anyone
	tx1 := getConflictsTx(smallNetFee)
	_, _, err = act.Send(tx1)
	fmt.Printf("tx1: %s\n", tx1.Hash().StringLE())
	checkNoErr(err)
	checkMempoolExactly(c, tx1.Hash())

	// tx2 conflicts with tx1 and has smaller netfee
	tx2 := getConflictsTx(smallNetFee-1, tx1.Hash())
	_, tx2VUB, err := act.Send(tx2)
	fmt.Printf("tx2: %s\n", tx2.Hash().StringLE())
	checkErrContains(err, hasConflictsErrText)
	checkMempoolExactly(c, tx1.Hash())

	// tx3 conflicts with mempooled tx1 and has larger netfee => tx1 should be replaced by tx3
	tx3 := getConflictsTx(smallNetFee+1, tx1.Hash())
	_, _, err = act.Send(tx3)
	fmt.Printf("tx3: %s\n", tx3.Hash().StringLE())
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash())

	// tx1 still does not conflicts with anyone, but tx3 is mempooled, conflicts with tx1
	// and has larger netfee => tx1 shouldn't be added again
	_, _, err = act.Send(tx1)
	checkErrContains(err, hasConflictsErrText)
	checkMempoolExactly(c, tx3.Hash())

	// tx2 can now safely be added because conflicting tx1 is not in mempool
	_, _, err = act.Send(tx2)
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash())

	// mempooled tx4 conflicts with tx5, but tx4 has smaller netfee => tx4 should be replaced by tx5 (Step 1, positive)
	tx5 := getConflictsTx(smallNetFee + 1)
	tx4 := getConflictsTx(smallNetFee, tx5.Hash())
	_, _, err = act.Send(tx4)
	fmt.Printf("tx4: %s\n", tx4.Hash().StringLE())
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash(), tx4.Hash())
	_, _, err = act.Send(tx5)
	fmt.Printf("tx5: %s\n", tx5.Hash().StringLE())
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash(), tx5.Hash())

	// multiple conflicts in attributes of single transaction: tx9 conflicts with tx6, tx7 and tx8.
	tx6 := getConflictsTx(smallNetFee*2 + 1)
	fmt.Printf("tx6: %s\n", tx6.Hash().StringLE())
	tx7 := getConflictsTx(smallNetFee)
	fmt.Printf("tx7: %s\n", tx7.Hash().StringLE())
	tx8 := getConflictsTx(smallNetFee)
	fmt.Printf("tx8: %s\n", tx8.Hash().StringLE())
	// need small network fee later
	tx9 := getConflictsTx(smallNetFee-2, tx6.Hash(), tx7.Hash(), tx8.Hash())
	_, _, err = act.Send(tx9)
	fmt.Printf("tx9: %s\n", tx9.Hash().StringLE())
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash(), tx5.Hash(), tx9.Hash())

	// multiple conflicts in attributes of multiple transactions: tx10 and tx11 conflict with tx6
	tx10 := getConflictsTx(smallNetFee, tx6.Hash())
	tx11 := getConflictsTx(smallNetFee, tx6.Hash())
	_, _, err = act.Send(tx10)
	fmt.Printf("tx10: %s\n", tx10.Hash().StringLE())
	checkNoErr(err)
	_, _, err = act.Send(tx11)
	fmt.Printf("tx11: %s\n", tx11.Hash().StringLE())
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash(), tx5.Hash(), tx9.Hash(), tx10.Hash(), tx11.Hash())

	// tx9 is in the mempool and conflicts with tx7, tx7 has larger network fee => tx7 should be added
	_, _, err = act.Send(tx7)
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash(), tx5.Hash(), tx7.Hash(), tx10.Hash(), tx11.Hash())

	// tx10 and tx11 conflict with tx6, tx6 has larger sum network fee => tx6 should be added
	_, _, err = act.Send(tx6)
	checkNoErr(err)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash(), tx5.Hash(), tx7.Hash(), tx6.Hash())

	// tx12 conflicts with tx2 and has larger network fee, but is not signed by tx2.Sender
	tx12 := getMaliciousTx(smallNetFee+5, tx2.Hash())
	_, _, err = maliciousAct.Send(tx12)
	checkErrContains(err, hasConflictsErrText)
	checkMempoolExactly(c, tx3.Hash(), tx2.Hash(), tx5.Hash(), tx7.Hash(), tx6.Hash())

	// Check the possible attack vectors described in the https://github.com/neo-project/neo/pull/2818#issuecomment-1568859533.
	checkAttack1 := func(mainFee int64, fail bool) {
		// mempooled txB1, txB2, txC conflict with txA
		txA := getConflictsTx(mainFee)
		txB1 := getConflictsTx(smallNetFee, txA.Hash())
		txB2 := getConflictsTx(smallNetFee, txA.Hash())
		txC := getMaliciousTx(smallNetFee, txA.Hash()) // malicious, thus, doesn't take into account during fee evaluation
		_, _, err = act.Send(txB1)
		checkNoErr(err)
		_, _, err = act.Send(txB2)
		checkNoErr(err)
		_, _, err = maliciousAct.Send(txC)
		checkNoErr(err)
		if fail {
			_, _, err = act.Send(txA)
			checkErrContains(err, hasConflictsErrText)
			checkMempoolContains(c, txB1.Hash(), txB2.Hash(), txC.Hash())
		} else {
			_, _, err = act.Send(txA)
			checkNoErr(err)
			checkMempoolDoesNotContain(c, txB1.Hash(), txB2.Hash(), txC.Hash())
		}
	}
	checkAttack1(smallNetFee*2, true)
	checkAttack1(smallNetFee*2+1, false)

	checkAttack2 := func(mainFee int64, fail bool) {
		// mempooled txB1, txB2, txB3 don't conflict with anyone, but txA conflicts with them
		txB1 := getConflictsTx(smallNetFee)
		txB2 := getConflictsTx(smallNetFee)
		txB3 := getConflictsTx(smallNetFee)
		txA := getConflictsTx(mainFee, txB1.Hash(), txB2.Hash(), txB3.Hash())
		_, _, err = act.Send(txB1)
		checkNoErr(err)
		_, _, err = act.Send(txB2)
		checkNoErr(err)
		_, _, err = act.Send(txB3)
		checkNoErr(err)
		if fail {
			_, _, err = act.Send(txA)
			checkErrContains(err, hasConflictsErrText)
			checkMempoolContains(c, txB1.Hash(), txB2.Hash(), txB3.Hash())
		} else {
			_, _, err = act.Send(txA)
			checkNoErr(err)
			checkMempoolDoesNotContain(c, txB1.Hash(), txB2.Hash(), txB3.Hash())
		}
	}
	checkAttack2(smallNetFee*3, true)
	checkAttack2(smallNetFee*3+1, false)

	// Start the on-chain txs test.
	fmt.Printf("\nStarting test for on-chain transactions.\n")

	checkMempoolContains(c, tx2.Hash())

	// Wait for the block to be processed and tx2 to be accepted.
	fmt.Printf("\nWaiting for the block to be processed and tx2 to be accepted...\n")
	aer, err := act.Wait(tx2.Hash(), tx2VUB, nil)
	checkNoErr(err)
	if aer.VMState != vmstate.Halt {
		panic("tx2 wasn't HALTed")
	}

	// Try to add tx1 to the mempool (tx2 is on chain, conflicts with tx1 and tx1 has smaller network fee).
	_, _, err = act.Send(tx1)
	checkErrContains(err, hasConflictsErrText)

	// tx13 confclits with on-chain tx2 => tx13 shouldn't be added to the pool.
	tx13 := getConflictsTx(smallNetFee, tx2.Hash())
	_, _, err = act.Send(tx13)
	checkErrContains(err, invalidAttributeErrTest)

	fmt.Printf("\nTest finished successfully.\n\n")
}

func waitForNewBlock(c *rpcclient.Client) {
	fmt.Println("Waiting for an empty block to be processed...")
	height, err := c.GetBlockCount()
	checkNoErr(err)
	fmt.Printf("Block count: %d\n", height)
	for {
		time.Sleep(5 * time.Second)
		newHeight, err := c.GetBlockCount()
		checkNoErr(err)
		fmt.Printf("Block count: %d\n", newHeight)
		if newHeight > height {
			break
		}
	}
}

func checkMempoolExactly(c *rpcclient.Client, txHashes ...util.Uint256) {
	checkMempool(c, true, true, txHashes...)
}

func checkMempoolContains(c *rpcclient.Client, txHashes ...util.Uint256) {
	checkMempool(c, false, true, txHashes...)
}

func checkMempoolDoesNotContain(c *rpcclient.Client, txHashes ...util.Uint256) {
	checkMempool(c, false, false, txHashes...)
}

func checkMempool(c *rpcclient.Client, exactly bool, contains bool, txHashes ...util.Uint256) {
	mp, err := c.GetRawMemPool()
	checkNoErr(err)

	mpMap := make(map[util.Uint256]struct{})
	for _, h := range mp {
		mpMap[h] = struct{}{}
	}

	for _, h := range txHashes {
		if _, ok := mpMap[h]; contains != ok {
			if contains {
				panic(fmt.Errorf("transaction %s not found in the mempool", h.StringLE()))
			} else {
				panic(fmt.Errorf("transaction %s is in the mempool, but expected not to be", h.StringLE()))
			}
		}
		delete(mpMap, h)
	}

	if exactly && len(mpMap) != 0 {
		panic("mempool contains unexpected items")
	}
}

func checkNoErr(err error) {
	if err != nil {
		panic(fmt.Errorf("no error expected, got %w", err))
	}
}

func checkErrContains(err error, expected string) {
	if err == nil {
		panic("error expected, got nil")
	}
	if !strings.Contains(err.Error(), expected) {
		panic(fmt.Errorf("error expected to contain `%s`, got `%s`", expected, err.Error()))
	}
}
