package schema

import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
	"reflect"
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
	Type HKey //the schema of object
}

func (h Href) ToObject(s *Container, o interface{}) {
	data, _ := s.Get(h.Hash)
	encoder.DeserializeRaw(data, o)
}

func (h Href) Expand(source *Container, info *HrefInfo) {
	if (source.Has(h.Hash)) {
		info.Has = append(info.Has, h.Hash)
		data, _ := source.Get(h.Hash)
		if (h.Type == EMPTY_KEY) {
			return
		}
		schema := Schema{}
		schemaData, _ := source.Get(h.Type)
		if (len(schemaData) == 0) {
			return
		}
		//fmt.Println(h.Type, schemaData)
		encoder.DeserializeRaw(schemaData, &schema)

		if (schema.StructName == "HArray") {
			var arr HArray
			encoder.DeserializeRaw(data, &arr)
			arr.Expand(source, info)
		} else {

			for i := 0; i < len(schema.StructFields); i++ {
				f := schema.StructFields[i]
				//fmt.Println("Href tag: ", reflect.StructTag(f.Tag).Get("href"))
				//fmt.Println("schemaData", schemaData, h.Type)
				if (string(f.Type) == "struct") {
					switch reflect.StructTag(f.Tag).Get("href") {
					case "object":
						href := Href{}
						encoder.DeserializeField(data, schema.StructFields, string(f.Name), &href)
						href.Expand(source, info)
					case "array":
						harray := HArray{}
						encoder.DeserializeField(data, schema.StructFields, string(f.Name), &harray)
						harray.Expand(source, info)
					}
				} else {
					//fmt.Println("type", string(f.Type), schema)
				}
			}
		}
	} else {
		if (h.Hash != EMPTY_KEY) {
			info.No = append(info.No, h.Hash)
		}
		if (h.Type != EMPTY_KEY) {
			info.No = append(info.No, h.Type)
		}
	}
}
