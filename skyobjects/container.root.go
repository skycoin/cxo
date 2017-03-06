package skyobjects

import (
	"github.com/evanlinjin/cxo/encoder"
	"github.com/skycoin/skycoin/src/cipher"
)

// SaveRoot saves a root object (if latest).
func (c *Container) SaveRoot(newRoot RootObject) (key cipher.SHA256) {
	// Only save the root if latest version.
	if newRoot.TimeStamp < c.rootTS {
		return
	}
	data := encoder.Serialize(newRoot)
	h := Href{SchemaKey: c._rootType, Data: data}
	return c.Save(h)
}

// GetCurrentRoot gets the latest local root as href.
func (c *Container) GetCurrentRoot() (h Href, e error) {
	if c.rootKey == (cipher.SHA256{}) {
		e = ErrorRootNotSpecified{}
		return
	}
	return c.Get(c.rootKey)
}

// GetCurrentRootDeserialised gets the latest local root object.
func (c *Container) GetCurrentRootDeserialised() (root RootObject, e error) {
	h, e := c.GetCurrentRoot()
	if e != nil {
		return
	}
	e = encoder.DeserializeRaw(h.Data, &root)
	return
}

// GetCurrentRootKey returns the local main root's key.
func (c *Container) GetCurrentRootKey() cipher.SHA256 { return c.rootKey }

// GetCurrentRootSequence returns the local main root's sequence.
func (c *Container) GetCurrentRootSequence() uint64 { return c.rootSeq }

// GetCurrentRootTimeStamp returns the local main root's time stamp.
func (c *Container) GetCurrentRootTimeStamp() int64 { return c.rootTS }
