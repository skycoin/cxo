package skyobject

type Age uint32

type Group struct {
	Name    string
	Leader  Reference  `skyobject:"schema=User"`
	Members References `skyobject:"schema=User"`
	Curator Dynamic
}

type User struct {
	Name   string ``
	Age    Age    `json:"age"`
	Hidden int    `enc:"-"`
}

type List struct {
	Name     string
	Members  References `skyobject:"schema=User"`
	MemberOf []Group
}

type Man struct {
	Name    string
	Age     Age
	Seecret []byte
	Owner   Group
	Friends List
}

type Bool bool
type Int8 int8
type Int16 int16
type Int32 int32
type Int64 int64
type Uint8 uint8
type Uint16 uint16
type Uint32 uint32
type Uint64 uint64
type Float32 float32
type Float64 float64
type String string
type Bytes []byte
