package internal

import (
	"compress/gzip"
	"context"
	"log"
	"os"

	"github.com/nspcc-dev/neo-go/pkg/io"
)

// WriteDump generates and writes the specific number of transactions to file.
func WriteDump(ctx context.Context, to string, opts BenchOptions) {
	out, err := os.Create(to)
	if err != nil {
		log.Printf("Something went wrong: %#v", err)
		os.Exit(2)
	}

	cp := gzip.NewWriter(out)
	defer func() {
		if err := cp.Flush(); err != nil {
			log.Fatalf("Could not flush buffer: %#v", err)
		}

		if err := cp.Close(); err != nil {
			log.Fatalf("Could not close compressor: %#v", err)
		}

		if err := out.Close(); err != nil {
			log.Fatalf("Could not close dump file: %#v", err)
		}
	}()

	rw := io.NewBinWriterFromIO(cp)
	opts.EncodeBinary(rw)

	Generate(ctx, opts, func(hash, blob string) error {
		rw.WriteString(hash)
		rw.WriteString(blob)

		return rw.Err
	})
}
