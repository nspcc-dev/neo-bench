package main

import (
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/crypto/hash"
	"github.com/nspcc-dev/neo-go/pkg/crypto/keys"
	"github.com/nspcc-dev/neo-go/pkg/io"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/nspcc-dev/neo-go/pkg/vm/emit"
)

type signer struct {
	script []byte
	addr   util.Uint160
	privs  []*keys.PrivateKey
	pubs   keys.PublicKeys
}

func newSigner(wifs ...string) (*signer, error) {
	var c signer
	for i := range wifs {
		priv, err := keys.NewPrivateKeyFromWIF(wifs[i])
		if err != nil {
			return nil, err
		}
		c.privs = append(c.privs, priv)
		c.pubs = append(c.pubs, priv.PublicKey())
	}
	var err error
	c.script, err = smartcontract.CreateMultiSigRedeemScript(len(c.pubs)/2+1, c.pubs)
	if err != nil {
		return nil, err
	}

	c.addr = hash.Hash160(c.script)
	return &c, nil
}

func (c *signer) signTx(txs ...*transaction.Transaction) {
	for _, tx := range txs {
		tx.Scripts = []transaction.Witness{{
			InvocationScript:   c.sign(tx),
			VerificationScript: c.script,
		}}
	}
}

func (c *signer) signBlock(b *block.Block) {
	b.Script.InvocationScript = c.sign(b)
	b.Script.VerificationScript = c.script
}

func (c *signer) sign(item hash.Hashable) []byte {
	h := hash.NetSha256(uint32(netmode.PrivNet), item)
	buf := io.NewBufBinWriter()
	for i := range c.privs {
		// It's kludgy, but we either sign for single node (1 out of 1)
		// or small private network (3 out of 4) and we need only 3
		// signatures for the latter case.
		if i == 3 {
			break
		}
		s := c.privs[i].SignHash(h)
		if len(s) != 64 {
			panic("wrong signature length")
		}
		emit.Bytes(buf.BinWriter, s)
	}
	return buf.Bytes()
}
