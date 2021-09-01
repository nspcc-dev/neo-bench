package main

import (
	"flag"
	"os"

	"github.com/nspcc-dev/neo-bench/internal"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
)

var (
	inp = flag.String("inp", "", "Path to read dump transactions.")
	out = flag.String("out", "./dump.txs", "Path to dump transactions.")
	cnt = flag.Int("cnt", 1_000_000, "Count of txs that would be generated.")
	typ = flag.String("type", internal.NEOTransfer, "Type of txs that would be generated.")
)

func main() {
	flag.Parse()

	ctx := internal.NewGracefulContext()

	switch {
	case inp != nil && *inp != "":
		internal.ReadDump(*inp)
	case out != nil && *out != "" && cnt != nil && *cnt > 0:
		p, _ := keys.NewPrivateKeyFromWIF("KxhEDBQyyEFymvfJD96q8stMbJMbZUb6D1PmXqBWZDU2WvbvVs9o")
		internal.WriteDump(ctx, *out, internal.BenchOptions{
			TransferType: *typ,
			TxCount:      uint64(*cnt),
			Senders:      []*keys.PrivateKey{p},
		})
	default:
		flag.PrintDefaults()
		os.Exit(0)
	}
}
