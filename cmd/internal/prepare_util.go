package internal

import (
	"github.com/nspcc-dev/neo-go/pkg/config/netmode"
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
	c.script, err = smartcontract.CreateDefaultMultiSigRedeemScript(c.pubs)
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

func (c *signer) sign(item hash.Hashable) []byte {
	h := hash.NetSha256(uint32(netmode.PrivNet), item)
	buf := io.NewBufBinWriter()
	need := smartcontract.GetDefaultHonestNodeCount(len(c.privs))
	for i := range c.privs {
		if i == need {
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
