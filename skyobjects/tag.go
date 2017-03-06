package skyobjects

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
	alias, ok := reflect.StructTag(tag).Lookup("skyobjects")
	if ok == false {
		e = fmt.Errorf("skyobjects not specified")
		return
	} else if alias == "" {
		return
	}

	// Extract data from 'skyobjects' tag.
	tags := strings.Split(alias, ",")
	for _, tag := range tags {
		switch {
		case tag == "href":
			st.href = true
		case strings.HasPrefix(tag, "schema_name="):
			st.schemaName = strings.TrimPrefix(tag, "schema_name=")
		}
	}

	// Fill aditional data from container.

	return
}
