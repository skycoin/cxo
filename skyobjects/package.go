package skyobjects

import "github.com/skycoin/skycoin/src/cipher"

// TypedDatas is an array of object data, packaged with their schemaKey.
type TypedDatas struct {
	Datas     [][]byte
	SchemaKey cipher.SHA256
}

// Package is good for sending/recieving a whole heap of data.
type Package struct {
	Found   []TypedDatas
	Missing []cipher.SHA256
}

// // PackagedKey represents an object's key, packaged with it's schema key.
// type PackagedKey struct {
// 	Key       cipher.SHA256
// 	SchemaKey cipher.SHA256
// }
//
// // PackagedKeyArray represents an array of object keys, packaged with a schema key.
// // All object keys included should be of type specified by the schema key.
// type PackagedKeyArray struct {
// 	Keys      []cipher.SHA256
// 	SchemaKey cipher.SHA256
// }
//
// // PackagedRoot represents a packaged root object.
// // All object keys needs to linked with a schema key.
// type PackagedRoot struct {
// 	Children  []PackagedKeyArray
// 	Sequence  uint64
// 	TimeStamp int64
// }
