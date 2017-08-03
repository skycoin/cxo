package skyobject

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node/log"
)

func TestPack_schemaOf(t *testing.T) {

	t.Skip("fuck the test")

	c := getCont()
	defer c.Close()

	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}
	pack, err := c.NewRoot(pk, sk, HashTableIndex|EntireTree,
		c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}
	pack.Append(&User{Name: "Alice", Age: 21})
}

func Test_anAverageTest(t *testing.T) {

	t.Skip("fuck the test")

	c := getCont()
	defer c.Close()
	pk, sk := cipher.GenerateKeyPair()
	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}
	pack, err := c.NewRoot(pk, sk, HashTableIndex|EntireTree,
		c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}
	pack.Append(&User{Name: "Alice", Age: 21})
	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}
	println(c.Inspect(pack.r))

	for _, name := range []string{"cxo.User", "cxo.Group", "cxo.Developer"} {
		if _, err := c.CoreRegistry().SchemaByName(name); err != nil {
			t.Error(err, name)
		}
	}

	typs := c.CoreRegistry().Types()

	for _, obj := range []interface{}{User{}, Group{}, Developer{}} {
		typ := typeOf(obj)
		if x, ok := typs.Inverse[typ]; !ok {
			t.Errorf("missing schema of %T called %q", typ, x)
		} else if xt, ok := typs.Direct[x]; !ok {
			t.Error("broken mapping")
		} else if xt != typ {
			t.Error("fucking, broken mapping")
		}
	}

	group := &Group{Name: "The Group"}

	pack.Append(group)
	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}
	println(c.Inspect(pack.r))

	println(fmt.Sprint(group.Leader.Short()))

	if err := group.Leader.SetValue(&User{Name: "Master"}); err != nil {
		t.Log(group.Leader)
		t.Fatal(err)
	}

	println(fmt.Sprint(group.Leader.Short()))

	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}
	println(c.Inspect(pack.r))

}

func Test_votes(t *testing.T) {

	t.Skip("fuck the test")

	type ContentVotes struct {
		Thread Refs `skyobject:"schema=Votes"`
		Post   Refs `skyobject:"schema=Votes"`
	}

	type Votes struct {
		Votes Refs `skyobject:"schema=Vote"`
	}

	type Vote struct {
		User   string
		Upvote bool
	}

	reg := NewRegistry(func(r *Reg) {
		r.Register("ContentVotes", ContentVotes{})
		r.Register("Votes", Votes{})
		r.Register("Vote", Vote{})
	})

	conf := NewConfig()
	conf.Registry = reg
	conf.MerkleDegree = 3

	if testing.Verbose() {
		conf.Log.Debug = true
		conf.Log.Pins = log.All
	}

	c := NewContainer(data.NewMemoryDB(), conf)

	//
	// create
	//

	pk, sk := cipher.GenerateKeyPair()

	if err := c.AddFeed(pk); err != nil {
		t.Fatal(err)
	}

	pack, err := c.NewRoot(pk, sk, HashTableIndex, c.CoreRegistry().Types())
	if err != nil {
		t.Fatal(err)
	}

	//

	threadVotes := new(Votes)
	postVotes := new(Votes)

	threadVotes.Votes = pack.Refs(
		&Vote{"Alex", true},
		&Vote{"Eva", false},
	)

	postVotes.Votes = pack.Refs(
		&Vote{"Bob", true},
		&Vote{"Tom", false},
	)

	fmt.Println("ThreadVotes", threadVotes.Votes.DebugString())
	fmt.Println("PostVotes", postVotes.Votes.DebugString())

	cv := new(ContentVotes)

	cv.Post = pack.Refs(threadVotes)
	cv.Thread = pack.Refs(postVotes)

	pack.Append(cv)

	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}

	fmt.Println("Inspect: ", c.Inspect(pack.Root()))

	votes := []interface{}{}

	for i := 0; i < 24; i++ {
		votes = append(votes, &Vote{fmt.Sprint(i), i%2 == 0})
	}

	threadVotes.Votes.Append(votes...)
	pack.SetRefByIndex(0, threadVotes)

	if _, err := pack.Save(); err != nil {
		t.Fatal(err)
	}
	fmt.Println("Inspect: ", c.Inspect(pack.Root()))

	fmt.Println("Inspect tree: ", threadVotes.Votes.DebugString())

	//
	// load
	//
}
