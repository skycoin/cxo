package skyobject

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

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
var testRegistry = registry.NewRegistry(func(r *registry.Reg) {
	// the name can be any, e.g. the "discovery.User" can be
	// "usr" or any other; feel free to choose
	r.Register("discovery.User", User{})
	r.Register("discovery.Feed", Feed{})
	r.Register("discovery.Post", Post{})
})

func getTestConfig() (c *Config) {
	c = NewConfig()
	c.InMemoryDB = true
	return
}

func getTestContainer() (c *Container) {
	var err error
	if c, err = NewContainer(getTestConfig()); err != nil {
		panic(err)
	}
	return
}

func assertNil(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func assertTrue(t *testing.T, v bool, msg string) {
	t.Helper()
	if v == false {
		t.Fatal(msg)
	}
}

func Test_fillinng(t *testing.T) {

	var (
		sc, rc = getTestContainer(), getTestContainer()
		pk, sk = cipher.GenerateKeyPair()
	)

	var up, err = sc.Unpack(sk, testRegistry)
	assertNil(t, err)

	var r = new(registry.Root)

	r.Pub = pk
	r.Nonce = 9021

	var usr = User{
		Name: "Alice",
		Age:  19,
	}

	var feed = Feed{
		Head: "Alices' feed",
		Info: "an average feed",
	}

	assertNil(t, sc.Save(up, r))

	//

}
