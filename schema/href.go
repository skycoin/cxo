package schema

import (
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

//type Href struct {
//	Type []byte        //the schema of object
//	Hash cipher.SHA256 //the hash of serialization of object
//}

type HrefStatic struct {
	Hash cipher.SHA256
}

type DataMap map[cipher.SHA256][]byte

//func CreateHref(data interface{}) Href {
//	schema := ExtractSchema(data)
//	structHash := cipher.SumSHA256(encoder.Serialize(data))
//	return Href{Type:encoder.Serialize(schema), Hash:structHash}
//}

func CreateHref(data interface{}) HrefStatic {
	structHash := cipher.SumSHA256(encoder.Serialize(data))
	return HrefStatic{Hash:structHash}
}

