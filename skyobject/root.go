package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

type Root struct {
	Time   int64
	Seq    uint64
	Schema Reference
	Object Reference

	// TODO
	Sign cipher.Sig    // signature
	Pub  cipher.PubKey // public key

	reg *schemaReg `enc:"-"` // TODO
}

func (r *Root) Touch() {
	r.Time = time.Now().UnixNano()
	r.Seq++
}
