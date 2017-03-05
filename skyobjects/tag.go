package skyobjects

import (
	"fmt"
	"reflect"
	"strings"
)

type skyTag struct {
	href   bool
	schema string
}

func getSkyTag(tag string) (st skyTag, e error) {
	alias, ok := reflect.StructTag(tag).Lookup("skyobjects")
	if ok == false {
		e = fmt.Errorf("skyobjects not specified")
		return
	} else if alias == "" {
		return
	}
	tags := strings.Split(alias, ",")
	for _, tag := range tags {
		switch {
		case tag == "href":
			st.href = true
		case strings.HasPrefix(tag, "type="):
			st.schema = strings.TrimPrefix(tag, "type=")
		}
	}
	return
}
