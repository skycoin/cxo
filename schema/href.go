package schema

import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)

var EMPTY_KEY HKey = HKey{}

type HKey cipher.SHA256

type HType struct {
	Type HKey
	Data HKey
}

func CreateKey(data []byte) HKey {
	return HKey(cipher.SumSHA256(data))
}

type HrefInfo struct {
	Has []HKey
	No  []HKey
}

type Href struct {
	Hash HKey
	Type []byte //the schema of object
}

func NewHref(key HKey, value interface{}) Href {
	schema := ExtractSchema(value)
	result := Href{Type:encoder.Serialize(schema)}
	result.Hash = key
	return result
}

func (h Href) ToObject(s *Store, o interface{}) {
	data, _ := s.Get(h.Hash)
	encoder.DeserializeRaw(data, o)
}

func (h Href) Expand(source *Store, info *HrefInfo) {
	if (source.Has(h.Hash)) {
		info.Has = append(info.Has, h.Hash)
		data, _ := source.Get(h.Hash)
		schema := Schema{}
		encoder.DeserializeRaw(h.Type, &schema)
		for i := 0; i < len(schema.StructFields); i++ {
			f := schema.StructFields[i]

			switch string(f.Type) {
			case "schema.Href":
				href := Href{}
				encoder.DeserializeField(data, schema.StructFields, string(f.Name), &href)
				href.Expand(source, info)
			case "schema.HArray":
				harray := HArray{}
				encoder.DeserializeField(data, schema.StructFields, string(f.Name), &harray)
				harray.Expand(source, info)
			}
		}
	} else {
		if (h.Hash != EMPTY_KEY) {
			info.No = append(info.No, h.Hash)
		}
	}
}
