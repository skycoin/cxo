package discovery

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
	Posts registry.Refs `skyobject:"schema=discovery.Post"` // posts
}

// A Post represnts post in the Feed
type Post struct {
	Head string // head of the Post
	Body string // content of the Post
}

// Registry that contains User, Feed and Post types
var Registry = registry.NewRegistry(func(r *registry.Reg) {
	// the name can be any, e.g. the "discovery.User" can be
	// "usr" or any other; feel free to choose
	r.Register("discovery.User", User{})
	r.Register("discovery.Feed", Feed{})
	r.Register("discovery.Post", Post{})
})
