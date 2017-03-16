package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

type Root struct {
	Time   uint64
	Seq    uint64
	Schema cipher.SHA256
	Object cipher.SHA256

	// TODO
	Sign cipher.Sig    // signature
	Pub  cipher.PubKey // public key

	reg *schemaReg `enc:"-"` // TODO
}

func (r *Root) Touch() {
	r.Time = time.Now().UnixNano()
	r.Seq++
}
