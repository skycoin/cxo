package schema

import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
	"reflect"
	"fmt"
)

var EMPTY_KEY cipher.SHA256 = cipher.SHA256{}

//type cipher.SHA256 cipher.SHA256

type HType struct {
	Type cipher.SHA256
	Data cipher.SHA256
}

func CreateKey(data []byte) cipher.SHA256 {
	return cipher.SHA256(cipher.SumSHA256(data))
}

type HrefInfo struct {
	Has []cipher.SHA256
	No  []cipher.SHA256
}

type HrefQuery struct {
	Items []cipher.SHA256
}

type Href struct {
	Hash cipher.SHA256
	Type cipher.SHA256 //the schema of object
}

func (h Href) ToObject(s *Container, o interface{}) {
	data, _ := s.Get(h.Hash)
	encoder.DeserializeRaw(data, o)
}

func (h Href) ExpandBySchema(source *Container, schemaKey cipher.SHA256, query *HrefQuery) {
	fmt.Println("ExpandBySchema")
	if (h.Type == schemaKey) {
		query.Items = append(query.Items, schemaKey)
		return
	}
	schema := Schema{}
	schemaData, _ := source.Get(h.Type)
	fmt.Println()
	if (len(schemaData) == 0) {
		return
	}
	encoder.DeserializeRaw(schemaData, &schema)

	data, _ := source.Get(h.Hash)
	if (h.Type == EMPTY_KEY) {
		return
	}
	if (schema.StructName == "HArray") {
		var arr HArray
		encoder.DeserializeRaw(data, &arr)
		arr.ExpandBySchema(source, schemaKey, query)
	} else {

		for i := 0; i < len(schema.StructFields); i++ {
			f := schema.StructFields[i]
			fmt.Println("schema.StructFields[i]", f)
			//fmt.Println("Href tag: ", reflect.StructTag(f.Tag).Get("href"))
			//fmt.Println("schemaData", schemaData, h.Type)
			if (string(f.Type) == "struct") {
				switch reflect.StructTag(f.Tag).Get("href") {
				case "object":
					href := Href{}
					encoder.DeserializeField(data, schema.StructFields, string(f.Name), &href)
					href.ExpandBySchema(source, schemaKey, query)
				case "array":
					harray := HArray{}
					encoder.DeserializeField(data, schema.StructFields, string(f.Name), &harray)
					harray.ExpandBySchema(source, schemaKey, query)
				}
			}
		}
	}

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
