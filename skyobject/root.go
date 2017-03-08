package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher/encoder"
)

type Root struct {
	Schema Schema
	Root   []byte
	Time   int64
	Seq    int64
}

func NewRoot(i interface{}) (root *Root) {
	return &Root{
		Schema: *getSchema(i),
		Root:   encoder.Serialize(i),
		Time:   time.Now().UnixNano(),
		Seq:    0,
	}
}

func (r *Root) Touch() {
	r.Seq++
	r.Time = time.Now().UnixNano()
}
