package skyobject

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	//"github.com/skycoin/skycoin/src/cipher/encoder"
)

func TestRoot_Encode(t *testing.T) {
	// Encode() []byte

	pk, sk := cipher.GenerateKeyPair()

	r := Root{
		Refs: []Dynamic{},
		Reg:  RegistryRef(cipher.SumSHA256([]byte("tyui"))),
		Pub:  pk,
		Seq:  890,
		Time: time.Now().Unix(),
		Sig:  cipher.SignHash(cipher.SumSHA256([]byte("ghjk")), sk),
		Hash: cipher.SumSHA256([]byte("qwer")),
		Prev: cipher.SumSHA256([]byte("asdf")),
	}

	bt := r.Encode()

	nr, err := DecodeRoot(bt)
	if err != nil {
		t.Fatal(err)
	}

	if len(nr.Refs) != 0 ||
		nr.Reg != r.Reg ||
		nr.Pub != pk ||
		nr.Seq != 890 ||
		nr.Time != r.Time ||
		nr.Sig != (cipher.Sig{}) ||
		nr.Hash != (cipher.SHA256{}) ||
		nr.Prev != r.Prev {
		t.Error("wrong")
	}

}

func TestRoot_Pack(t *testing.T) {
	// Pack() (rp *data.RootPack)

	// TODO (kostyarin): low priority

}

func TestContainer_PackToRoot(t *testing.T) {
	// PackToRoot(pk cipher.PubKey, rp *data.RootPack) (*Root, error)

	// TODO (kostyarin): low priority

}

func TestContainer_LastFullPack(t *testing.T) {
	// LastFullPack(pk cipher.PubKey) (rp *data.RootPack, err error)

	// TODO (kostyarin): low priority

}

func TestContainer_LastFull(t *testing.T) {
	// LastFull(pk cipher.PubKey) (r *Root, err error)

	// TODO (kostyarin): low priority

}

func TestContainer_LastPack(t *testing.T) {
	// LastPack(pk cipher.PubKey) (rp *data.RootPack, err error)

	// TODO (kostyarin): low priority

}

func TestContainer_Last(t *testing.T) {
	// Last(pk cipher.PubKey) (r *Root, err error)

	// TODO (kostyarin): low priority

}

func TestDecodeRoot(t *testing.T) {
	// DecodeRoot(val []byte) (r *Root, err error)

	t.Skip("joined with TestRoot_Encode")

}

func TestRoot_Short(t *testing.T) {
	// Short() string

	// TODO (kostyarin): low priority

}
func TestRoot_String(t *testing.T) {
	// String() string

	// TODO (kostyarin): low priority

}

func TestContainer_AddRoot(t *testing.T) {
	// AddRoot(pk cipher.PubKey, rp *data.RootPack) (r *Root, err error)

	// TODO (kostyarin): low priority

}

func TestContainer_MarkFull(t *testing.T) {
	// MarkFull(r *Root) (err error)

	// TODO (kostyarin): low priority

}

func TestContainer_RootBySeq(t *testing.T) {
	// RootBySeq(pk cipher.PubKey, seq uint64) (r *Root, err error)

	// TODO (kostyarin): low priority

}
