package node

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// A Container is wrapper around skyobject.Container.
// The container shares all modifications returning
// wrapped Root objects
type Container struct {
	*skyobject.Container
	node *Node
}

// Container returns wrapper of skyobejct.Container
func (s *Node) Container() *Container {
	return &Container{s.so, s}
}

// wrap *skyobject.Root
func (c *Container) wrapRoot(sr *skyobject.Root) *Root {
	return &Root{sr, c}
}

// NewRoot similar to (*skyobject.Container).NewRoot but it returns
// wrapped Root object
func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root,
	err error) {

	var sr *skyobject.Root
	if sr, err = c.Container.NewRoot(pk, sk); err != nil {
		return
	}
	r = c.wrapRoot(sr)
	return
}

// NewRootReg similar to (*skyobject.Container).NewRootReg
// but it returns wrapped Root object
func (c *Container) NewRootReg(pk cipher.PubKey, sk cipher.SecKey,
	rr skyobject.RegistryReference) (r *Root, err error) {

	var sr *skyobject.Root
	if sr, err = c.Container.NewRootReg(pk, sk, rr); err != nil {
		return
	}
	r = c.wrapRoot(sr)
	return
}

// AddRootPack wrapper
func (c *Container) AddRootPack(rp *data.RootPack) (r *Root,
	err error) {

	var sr *skyobject.Root
	if sr, err = c.Container.AddRootPack(rp); err != nil {
		return
	} else if sr != nil {
		r = c.wrapRoot(sr)
	}
	return

}

// LastRoot wrapper
func (c *Container) LastRoot(pk cipher.PubKey) (r *Root) {
	if sr := c.Container.LastRoot(pk); sr != nil {
		r = c.wrapRoot(sr)
	}
	return
}

// LastRootSk wrapper
func (c *Container) LastRootSk(pk cipher.PubKey, sk cipher.SecKey) (r *Root) {
	if sr := c.Container.LastRootSk(pk, sk); sr != nil {
		r = c.wrapRoot(sr)
	}
	return
}

// LastFullRoot wrapper
func (c *Container) LastFullRoot(pk cipher.PubKey) (r *Root) {
	if sr := c.Container.LastFullRoot(pk); sr != nil {
		r = c.wrapRoot(sr)
	}
	return
}

// RootByHash wrapper
func (c *Container) RootByHash(rr skyobject.RootReference) (r *Root, ok bool) {
	var sr *skyobject.Root
	if sr, ok = c.Container.RootByHash(rr); sr != nil {
		r = c.wrapRoot(sr)
	}
	return
}

// A Root represents wrapper of *skyobject.Root that
// shares all changes
type Root struct {
	*skyobject.Root
	c *Container
}

// send changes to all subscribers
func (r *Root) send(rp data.RootPack) {
	if !r.c.node.hasFeed(r.Pub()) {
		return // don't send
	}
	r.c.node.sendToFeed(r.Pub(), r.c.node.src.NewRootMsg(
		r.Pub(), // feed
		rp,      // root pack
	), nil)
}

// Touch wrapper
func (r *Root) Touch() (rp data.RootPack, err error) {
	if rp, err = r.Root.Touch(); err == nil {
		r.send(rp)
	}
	return
}

// Inject wrapper
//
// Deprecated: use Append instead
func (r *Root) Inject(schemName string, i interface{}) (inj skyobject.Dynamic,
	rp data.RootPack, err error) {

	if inj, rp, err = r.Root.Inject(schemName, i); err == nil {
		r.send(rp)
	}
	return
}

// InjectMany wrapper
//
// Deprecated: use Append instead
func (r *Root) InjectMany(schemaName string,
	i ...interface{}) (injs []skyobject.Dynamic, rp data.RootPack,
	err error) {

	if injs, rp, err = r.Root.InjectMany(schemaName, i...); err == nil {
		r.send(rp)
	}
	return
}

// Append wrapper
func (r *Root) Append(drs ...skyobject.Dynamic) (rp data.RootPack, err error) {
	if rp, err = r.Root.Append(drs...); err == nil {
		r.send(rp)
	}
	return
}

// Repalce wrapper
func (r *Root) Replace(refs []skyobject.Dynamic) (rp data.RootPack, err error) {
	if rp, err = r.Root.Replace(refs); err == nil {
		r.send(rp)
	}
	return
}

// Walker returns RootWalker of the Root
func (r *Root) Walker() (w *RootWalker) {
	return NewRootWalker(r)
}
