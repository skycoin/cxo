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
