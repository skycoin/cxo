package skyobjects

import (
	"github.com/skycoin/cxo/data"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func (c *Container) dbSave(ds *data.DB, schemaKey cipher.SHA256, data []byte) (key cipher.SHA256) {
	h := href{Type: schemaKey, Data: data}
	key = ds.AddAutoKey(encoder.Serialize(h))
	return
}

func (c *Container) dbSaveSchema(ds *data.DB, data []byte) (schemaKey cipher.SHA256) {
	h := href{Type: _schemaType, Data: data}
	return ds.AddAutoKey(encoder.Serialize(h))
}

func (c *Container) dbGet(ds *data.DB, key cipher.SHA256) (ref *href) {
	ref = &href{}
	data, ok := ds.Get(key)
	if ok == false {
		return
	}
	encoder.DeserializeRaw(data, ref)
	return
}
