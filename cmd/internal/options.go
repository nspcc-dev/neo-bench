package internal

import (
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/io"
)

// BenchOptions describes transactions contained in a dump.
type BenchOptions struct {
	TransferType string
	TxCount      uint64
	ToCount      int
	Senders      []*keys.PrivateKey
}

func (o *BenchOptions) EncodeBinary(w *io.BinWriter) {
	w.WriteString(o.TransferType)
	w.WriteVarUint(uint64(len(o.Senders)))
	for _, p := range o.Senders {
		w.WriteBytes(p.Bytes())
	}
	w.WriteVarUint(uint64(o.ToCount))
	w.WriteU64LE(uint64(o.TxCount))
}

func (o *BenchOptions) DecodeBinary(r *io.BinReader) {
	o.TransferType = r.ReadString()
	privCount := int(r.ReadVarUint())
	if r.Err != nil {
		return
	}

	o.Senders = make([]*keys.PrivateKey, privCount)
	buf := make([]byte, 32)
	for i := range o.Senders {
		r.ReadBytes(buf)
		p, err := keys.NewPrivateKeyFromBytes(buf)
		if err != nil {
			r.Err = err
			return
		}
		o.Senders[i] = p
	}

	o.ToCount = int(r.ReadVarUint())
	o.TxCount = r.ReadU64LE()
}
