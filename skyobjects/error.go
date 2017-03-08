package skyobjects

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

// ErrorSchemaNotFound occurs when the Container doesn't have a schema of key/name.
type ErrorSchemaNotFound struct {
	SchemaKey  cipher.SHA256
	SchemaName string
}

func (e ErrorSchemaNotFound) Error() string {
	switch {
	case e.SchemaKey != cipher.SHA256{} && e.SchemaName == "": // Error from key.
		return fmt.Sprintf("schema of key `%s` not found", e.SchemaKey.Hex())
	case e.SchemaKey == cipher.SHA256{} && e.SchemaName != "": // Error from name.
		return fmt.Sprintf("schema of name `%s` not found", e.SchemaName)
	}
	return "schema not found"
}

// ErrorKeyIsNotSchema occurs when specified key does not reference a schema.
type ErrorKeyIsNotSchema struct {
	SchemaKey cipher.SHA256
}

func (e ErrorKeyIsNotSchema) Error() string {
	return fmt.Sprintf("specified key '%s' does not reference a schema", e.SchemaKey)
}

// ErrorSkyObjectNotFound occurs when specified key does not reference anything
// in the container.
type ErrorSkyObjectNotFound struct {
	Key cipher.SHA256
}

func (e ErrorSkyObjectNotFound) Error() string {
	return fmt.Sprintf("skyobject of key '%s' not found", e.Key.Hex())
}

// ErrorRootNotSpecified occurs when there the main root object is not specified
// in the container.
type ErrorRootNotSpecified struct{}

func (e ErrorRootNotSpecified) Error() string {
	return "root not specified"
}
