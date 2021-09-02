package main

import (
	"flag"
	"os"

	"github.com/nspcc-dev/neo-bench/internal"
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
		internal.WriteDump(ctx, *out, *typ, *cnt)
	default:
		flag.PrintDefaults()
		os.Exit(0)
	}
}
