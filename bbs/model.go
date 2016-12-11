package bbs

import (
	"github.com/skycoin/cxo/skyobject"
)

//type BoardType skyobject.HashLink
//

type Board struct {
	Name    string
	Threads skyobject.HashSlice
}

type Thread struct {
	Name  string
	Posts skyobject.HashSlice
}

type Post struct {
	Header string
	Text   string
	Poster skyobject.HashLink
}

type Poster struct {
	data []byte
}



//func (m *BoardType) Save(c skyobject.ISkyObjects) skyobject.Href {
//	return (*skyobject.HashLink)(m).Save(c)
//}
//
//func (m *BoardType) SetData(data []byte) {
//	(*skyobject.HashLink)(m).SetData(data)
//}
//
//func (m *BoardType) References(c skyobject.ISkyObjects) []cipher.SHA256 {
//	return (*skyobject.HashLink)(m).References(c)
//}
//
//func (m *BoardType) Type() cipher.SHA256 {
//	return (*skyobject.HashLink)(m).Type()
//}
//
//func (m *BoardType) String(c skyobject.ISkyObjects) string {
//	return (*skyobject.HashLink)(m).String(c)
//}


