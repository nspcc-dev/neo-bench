package internal

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"log"
	"time"

	"github.com/CityOfZion/neo-go/pkg/core/transaction"
	"github.com/CityOfZion/neo-go/pkg/crypto/keys"
	"github.com/CityOfZion/neo-go/pkg/encoding/address"
	"github.com/CityOfZion/neo-go/pkg/io"
	"github.com/CityOfZion/neo-go/pkg/rpc"
)

type (
	// Dump contains hashes and marshaled transactions.
	Dump struct {
		Hashes       map[string]struct{}
		Transactions []string
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
	fromAddress := wif.PrivateKey.Address()
	fromAddressHash, err := address.StringToUint160(fromAddress)
	if err != nil {
		log.Fatalf("could not fetch address: %#v", err)
	}

	tx := &transaction.Transaction{
		Type:    transaction.InvocationType,
		Version: 0,
		Data: &transaction.InvocationTX{
			Script:  []byte{0x51},
			Gas:     0,
			Version: 0,
		},
		Attributes: []transaction.Attribute{},
		Inputs:     []transaction.Input{},
		Outputs:    []transaction.Output{},
		Scripts:    []transaction.Witness{},
		Trimmed:    false,
	}
	tx.Attributes = append(tx.Attributes,
		transaction.Attribute{
			Usage: transaction.Description,
			Data:  make([]byte, 16),
		})
	tx.Attributes = append(tx.Attributes,
		transaction.Attribute{
			Usage: transaction.Script,
			Data:  fromAddressHash.BytesBE(),
		})
	return tx
}

// Generate used to generate the specified number of transactions.
func Generate(ctx context.Context, count int, callback ...GenerateCallback) *Dump {
	start := time.Now()

	dump := Dump{
		Hashes:       make(map[string]struct{}, count),
		Transactions: make([]string, 0, count),
	}

	log.Printf("Generate %d txs", count)

	wif, err := getWif()
	if err != nil {
		log.Fatalf("Could not get wif: %v", err)
	}

	buf := io.NewBufBinWriter()

	tx := newTX(wif)
	for i := 0; i < count; i++ {
		if ctx.Err() != nil {
			log.Fatal(ctx.Err())
		}

		tx := *tx
		binary.BigEndian.PutUint64(tx.Attributes[0].Data, uint64(i))

		if err := rpc.SignTx(&tx, wif); err != nil {
			log.Fatalf("Could not read random bytes")
		}

		tx.EncodeBinary(buf.BinWriter)

		if buf.Err != nil {
			log.Fatalf("Could not prepare transaction: %d %v", i, err)
		}

		hash := tx.Hash().String()
		blob := hex.EncodeToString(buf.Bytes())

		dump.Hashes[hash] = struct{}{}
		dump.Transactions = append(dump.Transactions, blob)

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
