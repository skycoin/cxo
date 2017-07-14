// Package data represents CXO database. The package includes in-memory
// database and on-drive database. Both databases implements the same interface.
//
// The DB is ACID and uses transactions, that can be rolled back.
// There are read-only and read-write transactions. See docs for
// DB.View, DB.Update and Tv, Tu. The Tv is read-only transaction
// and Tu is read-write transaction.
//
// Any transaction allows access to Objects, Feeds and Roots (through Feeds).
// Approx. schema is:
//
//     objects { key -> value }
//     feeds   { pk -> roots { seq -> root } }
//
//
// Objects. There are ViewObjects and UpdateObjects interfaces that used
// to manipulate Objects. The first is view only, the second is for
// modifications (and viewing too). Read-only transaction returns ViewObjects,
// read-write transaction returns UpdateObjects. Thus, you will never
// modify any read-only transaction.
//
// TODO (kostyarin) improve the docs
package data
