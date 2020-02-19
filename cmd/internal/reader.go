package internal

import (
	"compress/gzip"
	"log"
	"os"
	"time"

	"github.com/CityOfZion/neo-go/pkg/io"
)

// ReadDump used to open dump of transactions.
func ReadDump(from string) *Dump {
	in, err := os.Open(from)
	if err != nil {
		log.Printf("Could not open dump file: %#v", err)
		os.Exit(2)
	}

	cp, err := gzip.NewReader(in)
	if err != nil {
		log.Printf("Could not prepare decompressor: %#v", err)
		os.Exit(2)
	}

	defer func() {
		if err := cp.Close(); err != nil {
			log.Fatalf("could not close decompressor: %#v", err)
		}

		if err := in.Close(); err != nil {
			log.Fatalf("could not close dump file: %#v", err)
		}
	}()

	rd := io.NewBinReaderFromIO(cp)
	count := rd.ReadU64LE()

	dump := &Dump{
		Hashes:       make(map[string]struct{}, count),
		Transactions: make([]string, 0, count),
	}

	start := time.Now()
	log.Printf("Read %d txs from %s", count, in.Name())
	for i := uint64(0); i < count; i++ {
		hash := rd.ReadString() // hash
		blob := rd.ReadString() // blob

		if rd.Err != nil {
			log.Fatalf("Could not read tx: %d %v", i, rd.Err)
		}

		dump.Hashes[hash] = struct{}{}
		dump.Transactions = append(dump.Transactions, blob)
	}

	log.Printf("Done %s", time.Since(start))
	return dump
}
