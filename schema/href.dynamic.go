package schema

import (
	"github.com/skycoin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)

type HDynamic struct {
	Href
	Schema []byte //the schema of object
}

func HrefDynamic(key cipher.SHA256, value interface{}) HDynamic {
	schema := ExtractSchema(value)
	result := HDynamic{Schema:encoder.Serialize(schema)}
	result.Hash = key
	return result
}

func (h *HDynamic) GetSchema() StructSchema{
	var result StructSchema
	encoder.DeserializeRaw(h.Schema, &result)
	return result
}
