package skyobject_test

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

type User struct {
	Name   string
	Age    uint32
	Hidden []byte `enc:"-"`
}

type Group struct {
	Name    string
	Leader  skyobject.Reference  `skyobject:"schema=test.User"`
	Members skyobject.References `skyobject:"schema=test.User"`
	Curator skyobject.Dynamic
}

type Developer struct {
	Name   string
	GitHub string
}

func Example() {
	reg := skyobject.NewRegistry()

	reg.Register("test.User", User{})
	reg.Register("test.Group", Group{})
	reg.Register("test.Developer", Developer{})

	// Create container associated with given
	// registry. The registry will be used
	// as registry of all root obejcts
	// created by the container. The
	// NewContainer calls (*Registry).Done
	// implicitly
	c := skyobject.NewContainer(data.NewMemoryDB(), reg)

	pk, sk := cipher.GenerateKeyPair()

	// Create empty detached root object
	root, _ := c.NewRoot(pk, sk)

	// callings of
	//  - Touch
	//  - Append
	//  - Replace
	// (1) increment seq number of the root
	// (2) set its timestamp to now
	// (3) sign the root
	// (4) attach the root to the container

	// thus every of the callings create new version
	// of the root

	root.Append(root.MustDynamic("test.Group", Group{
		Name: "group #1",
		Leader: root.Save(User{
			Name:   "Bob Marley",
			Age:    20,
			Hidden: []byte("secret"),
		}),
		Members: root.SaveArray(
			User{"Alex", 15, nil},
			User{"Alice", 16, nil},
			User{"Eva", 21, nil},
		),
		Curator: root.MustDynamic("test.Developer", Developer{
			Name:   "kostyarin",
			GitHub: "logrusorgru",
		}),
	}))

	// TODO

}
