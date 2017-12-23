// Package intro used for examples. The
// package implements some exampe types and
// registry with these types
package intro

import (
	"github.com/skycoin/cxo/skyobject/registry"
)

// A User's information
type User struct {
	Name string // user's name
	Age  uint32 // user's age
}

// A Feed of the User
type Feed struct {
	Head  string        // head of the feed
	Info  string        // brief information about the feed
	Posts registry.Refs `skyobject:"schema=intro.Post"` // posts
}

// A Post represnts post in the Feed
type Post struct {
	Head string // head of the Post
	Body string // content of the Post
}

// Registry that contains User, Feed and Post types
var Registry = registry.NewRegistry(func(r *registry.Reg) {
	// the name can be any, e.g. the "intro.User" can be
	// "usr" or any other; feel free to choose
	r.Register("intro.User", User{})
	r.Register("intro.Feed", Feed{})
	r.Register("intro.Post", Post{})
})
