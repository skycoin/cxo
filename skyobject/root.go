package skyobject

import (
	"time"

	"github.com/skycoin/skycoin/src/cipher"
)

type Root struct {
	Time   time.Time
	Seq    int64
	Schema cipher.SHA256
	Root   cipher.SHA256
}
