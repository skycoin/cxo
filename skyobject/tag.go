package skyobject

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/skycoin/skycoin/src/cipher"
)

type skyTag struct {
	href       bool
	schemaName string
	schemaKey  cipher.SHA256
}

func getSkyTag(c *Container, tag string) (st skyTag, e error) {
	alias, ok := reflect.StructTag(tag).Lookup("skyobject")
	if ok == false {
		e = fmt.Errorf("skyobject not specified")
		return
	} else if alias == "" {
		return
	}
	// Extract data from 'skyobject' tag.
	tags := strings.Split(alias, ",")
	for _, tag := range tags {
		switch {
		case tag == "href":
			st.href = true
		case strings.HasPrefix(tag, "schema_name="):
			st.schemaName = strings.TrimPrefix(tag, "schema_name=")
		}
	}
	// Fill additional data to skyTag object from container.
	schema, e := c.GetSchemaOfName(st.schemaName)
	if e != nil {
		return
	}
	st.schemaName = schema.Name
	return
}
