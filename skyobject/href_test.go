package skyobject

import (
	"testing"
	"github.com/skycoin/cxo/data"
)

type TestHref1 struct {
	Field1       uint32
	Field2       bool
	fieldPrivate bool
}


type TestHrefObj struct {
	Field1 uint32
	Field2 HashObject
	Field3 bool
}

var referenceTypesCount = 4

func Test_Href_Object(T *testing.T) {
	db := data.NewDB()
	container := SkyObjects(db)

	objects := container.ds.Statistic().Total
	if objects != referenceTypesCount {
		T.Fatal("Wrong objects count")
	}

	var t1 TestHref1
	t1.Field1 = 255
	t1.Field2 = false
	//
	h1 := NewObject(t1)
	container.Save(&h1)
	objects = container.ds.Statistic().Total - referenceTypesCount
	if objects != 2 {
		T.Fatal("Wrong objects count")
	}

	var to TestHrefObj
	to.Field1 = 255
	to.Field2 = h1
	to.Field3 = false

	ho := NewObject(to)
	container.Save(&ho)
	objects = container.ds.Statistic().Total - referenceTypesCount
	if objects != 4 {
		T.Fatal("Wrong objects count")
	}
}
