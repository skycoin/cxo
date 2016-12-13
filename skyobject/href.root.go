package skyobject

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

var _rootSchemaKey cipher.SHA256 = cipher.SumSHA256(encoder.Serialize(*ReadSchema(HashRoot{})))

type hashRoot struct {
	Root Href
	Size int32
	Sign cipher.Sig
	owner *cipher.SecKey `enc:"-"`
}

type HashRoot HashLink

func newRoot(ref Href, owner *cipher.SecKey) HashRoot {
	res := HashRoot{value:hashRoot{Root:ref, owner:owner}}
	return res
}

//SetData for internal usage
func (h *HashRoot) SetData(data []byte) {
	h.rdata = data
}

func (h *HashRoot) Type() cipher.SHA256 {
	return _rootSchemaKey
}

func (h *HashRoot) References(c ISkyObjects) RefInfoMap {
	value := hashRoot{}
	encoder.DeserializeRaw(h.rdata, &value)
	return value.Root.References(c)
}

func (h *HashRoot) save(c ISkyObjects) Href {
	value := h.value.(hashRoot)
	for _, s := range value.Root.References(c) {
		value.Size += s
	}

	value.Sign = cipher.SignHash(h.Ref, *value.owner)
	h.value = value
	return ((*HashLink)(h)).save(c)
}

func (h *HashRoot) String(c ISkyObjects) string {
	return ""
}
