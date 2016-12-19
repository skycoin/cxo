package skyobject

import (
	"testing"
	"github.com/skycoin/cxo/data"
	"fmt"
)

type TestHref1 struct {
	Field1       uint32
	Field2       bool
	fieldPrivate bool
}

type TestHref2 struct {
	Field1 uint32
	Field2 HashLink
	Field3 bool
}

type TestHref3 struct {
	Field1 uint32
	Field2 HashLink
	Field3 bool
	Field4 HashArray
}

type TestHrefObj struct {
	Field1 uint32
	Field2 HashObject
	Field3 bool
}

var referenceTypesCount = 4

func Test_Href_1(T *testing.T) {
	db := data.NewDB()
	container := SkyObjects(db)

	objects := container.ds.Statistic().Total
	if objects != referenceTypesCount {
		T.Fatal("Wrong objects count", objects)
	}

	var t1 TestHref1
	t1.Field1 = 255
	t1.Field2 = false
	//
	h1 := NewLink(t1)
	container.Save(&h1)
	objects = container.ds.Statistic().Total - referenceTypesCount
	if objects != 1 {
		T.Fatal("Wrong objects count", objects)
	}

	var t2 TestHref1
	t2.Field1 = 15
	t2.Field2 = true

	h2 := NewLink(t2)
	container.Save(&h2)
	objects = container.ds.Statistic().Total - referenceTypesCount

	fmt.Println(container.Statistic())
	if objects != 2 {
		T.Fatal("Wrong objects count", objects)
	}
}

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
