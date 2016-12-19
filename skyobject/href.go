package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
	"fmt"
)

var  _schemaType cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(Schema{}))

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

type RefInfoMap map[cipher.SHA256]int32

type IHashObject interface {
	save(ISkyObjects) Href
	SetData(data []byte)
	References(c ISkyObjects) RefInfoMap
	Type() cipher.SHA256
	String(c ISkyObjects) string
}

func (s *Href) References(c ISkyObjects) RefInfoMap {
	result := RefInfoMap{}
	rdata, _ := c.Get(s.Ref)
	result[s.Ref] = int32(len(rdata))
	ref := href{}
	encoder.DeserializeRaw(rdata, &ref)
	hobj, ok := c.HashObject(ref.Type, ref.Data)
	var childRefs RefInfoMap
	if (!ok){
		hobj = &HashObject{rdata:s.Ref[:]}
	}

	childRefs = hobj.References(c)

	return mergeRefs(result, childRefs)
}

func (s *Href) String(c ISkyObjects) string {
	rdata, _ := c.Get(s.Ref)
	ref := href{}
	encoder.DeserializeRaw(rdata, &ref)
	hobj, _ := c.HashObject(ref.Type, ref.Data)
	return hobj.String(c)
}

func (h *HashLink) Fields(key cipher.SHA256) (map[string]string) {
	return map[string]string{}
}

func mergeRefs(res RefInfoMap, batch RefInfoMap) RefInfoMap {
	for h, s := range batch {
		res[h] = s
	}
	return res
}

func (rfs *RefInfoMap) String() string {
	var sum int32 = 0
	for _, s := range *rfs {
		sum += s
	}
	return fmt.Sprint("Total objects:", len(*rfs), " Size:", sum)
}
