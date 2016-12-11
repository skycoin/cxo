package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

type Href struct {
	Ref   cipher.SHA256
	rdata []byte        `enc:"-"`
	rtype []byte        `enc:"-"`
	value interface{}   `enc:"-"`
}

type href struct {
	Type cipher.SHA256
	Data []byte
}

type IHashObject interface {
	Save(ISkyObjects) Href
	SetData(data []byte)
	References(c ISkyObjects) []cipher.SHA256
	Type() cipher.SHA256
	String(c ISkyObjects) string
}

func (s *Href) References(c ISkyObjects) []cipher.SHA256 {
	rdata, _ := c.Get(s.Ref)
	ref := href{}
	encoder.DeserializeRaw(rdata, &ref)
	hobj, _ := c.HashObject(ref.Type, ref.Data)
	return hobj.References(c)
}

func (s *Href) String(c ISkyObjects) string {
	rdata, _ := c.Get(s.Ref)
	ref := href{}
	encoder.DeserializeRaw(rdata, &ref)
	hobj, _ := c.HashObject(ref.Type, ref.Data)
	return hobj.String(c)
}
