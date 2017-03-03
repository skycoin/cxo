SkyObjects API
===

## TODO: Features to implement

### (Container).SetDB(\*data.DB)

Sets a new DB for container.

Remember to set ...
```
rootKey cipher.SHA256
rootSeq uint64
schemas map[cipher.SHA256]string
```
... to appropriate values.

### (Container).GetReferences(schemaKey cipher.SHA256, objData []byte) map[cipher.SHA256]cipher.SHA256

Gets list of objects that reference the specified object.

Returns a map of:
* Key: key of object stored
* Value: schemaKey of object stored

### (RootObject).GetDescendants(c \*Container) map[cipher.SHA256]bool

Gets all keys of descendants of a root object.

Boolean value:
* TRUE: We have a copy of this object in container.
* FALSE: We don't have a copy of this object in container.
