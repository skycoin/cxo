package skyobject

import (
	"testing"
	"github.com/skycoin/cxo/data"
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
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
	Field4 HashSlice
}

var referenceTypesCount = 3

func Test_Href_1(T *testing.T) {
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
	h1 := NewLink(t1)
	container.Save(&h1)
	objects = container.ds.Statistic().Total - referenceTypesCount
	if objects != 3 {
		T.Fatal("Wrong objects count")
	}

	var t2 TestHref1
	t2.Field1 = 15
	t2.Field2 = true

	h2 := NewLink(t2)
	container.Save(&h2)
	objects = container.ds.Statistic().Total - referenceTypesCount

	fmt.Println(container.Statistic())
	if objects != 5 {
		T.Fatal("Wrong objects count")
	}
}

func Test_Href_2(T *testing.T) {
	db := data.NewDB()
	container := SkyObjects(db)

	var t1 TestHref1
	t1.Field1 = 255
	t1.Field2 = false

	var t2 TestHref1
	t2.Field1 = 22
	t1.Field2 = true

	var t3 TestHref1
	t3.Field1 = 33
	t3.Field2 = false

	h1 := NewLink(t1)
	h2 := NewLink(t2)
	h3 := NewLink(t3)

	container.Save(&h1)
	container.Save(&h2)
	container.Save(&h3)

	objects := container.ds.Statistic().Total - referenceTypesCount

	fmt.Println(container.Statistic())
	if objects != 7 {
		T.Fatal("Wrong objects count", objects)
	}

	var t4 TestHref1
	t4.Field1 = 44
	t4.Field2 = false

	h4 := NewSlice(TestHref1{}, t2, t3, t4)

	container.Save(&h4)
	objects = container.ds.Statistic().Total - referenceTypesCount
	if objects != 8 {
		T.Fatal("Wrong objects count", objects)
	}

	var ta TestHref3
	ta.Field1 = 11
	ta.Field3 = true
	ta.Field2 = h1
	ta.Field4 = h4

	ha := NewLink(ta)

	rr := container.Save(&ha)
	refs := rr.References(container)
	fmt.Println(refs.String())
}

func Test_Href_Root(T *testing.T) {
	db := data.NewDB()
	container := SkyObjects(db)

	var t1 TestHref1
	t1.Field1 = 255
	t1.Field2 = false
	h1 := NewLink(t1)
	container.Save(&h1)

	_, secKey := cipher.GenerateKeyPair()

	rh := newRoot(Href(h1), &secKey)
	r := container.Save(&rh)
	refs := r.References(container)

	fmt.Println(refs.String())
	fmt.Println("Statistic", container.Statistic())
}
