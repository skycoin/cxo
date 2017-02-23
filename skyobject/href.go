package skyobject

import (
	"fmt"
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)

var _schemaType cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(Schema{}))

type Href struct {
	Ref   cipher.SHA256
	rdata []byte      `enc:"-"`
	rtype []byte      `enc:"-"`
	value interface{} `enc:"-"`
}

type href struct {
	Type cipher.SHA256
	Data []byte
}

type RefInfoMap map[cipher.SHA256]int32

type IHashObject interface {
	save(ISkyObjects) Href
	SetData(tp []byte, data []byte)
	References(c ISkyObjects) RefInfoMap
	Type() cipher.SHA256
	String(c ISkyObjects) string
}

func (s *Href) References(c ISkyObjects) RefInfoMap {
	result := RefInfoMap{}
	data, _ := c.Get(s.Ref)

	ref := href{}
	encoder.DeserializeRaw(data, &ref)

	hobj, ok := c.HashObject(ref)
	var childRefs RefInfoMap
	if !ok {
		//fmt.Println("Not a hash object")
		schemaData, _ := c.Get(ref.Type)
		smref := href{}
		encoder.DeserializeRaw(schemaData, &smref)
		hobj = &HashObject{rdata: ref.Data, rtype: smref.Data}
	}

	//fmt.Println("href", ref)
	childRefs = hobj.References(c)
	//fmt.Println(childRefs)

	result[s.Ref] = int32(len(data))
	return mergeRefs(result, childRefs)
}

func (s *Href) Children(c ISkyObjects) (result RefInfoMap) {
	var ref = c.GetRef(s.Ref)
	hobj, _ := c.HashObject(ref)
	result = hobj.References(c)
	return
}

func (s *Href) String(c ISkyObjects) string {
	//rdata, _ := c.Get(s.Ref)
	//ref := href{}
	//encoder.DeserializeRaw(rdata, &ref)
	//hobj, _ := c.HashObject(ref)
	return s.Ref.Hex()
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
