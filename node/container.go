package node

import (
	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

type Container struct {
	*skyobject.Container
	client *Client
}

func (c *Container) wrapRoot(sr *skyobject.Root) *Root {
	return &Root{sr, c}
}

func (c *Container) NewRoot(pk cipher.PubKey, sk cipher.SecKey) (r *Root,
	err error) {

	var sr *skyobject.Root
	if sr, err = c.Container.NewRoot(pk, sk); err != nil {
		return
	}
	r = &Root{sr, c}
	return
}

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

func (c *Container) LastRoot(pk cipher.PubKey) (r *Root) {
	if sr := c.Container.LastRoot(pk); sr != nil {
		r = c.wrapRoot(sr)
	}
	return
}

func (c *Container) LastRootSk(pk cipher.PubKey, sk cipher.SecKey) (r *Root) {
	if sr := c.Container.LastRootSk(pk, sk); sr != nil {
		r = c.wrapRoot(sr)
	}
	return
}

func (c *Container) LastFullRoot(pk cipher.PubKey) (r *Root) {
	if sr := c.Container.LastFullRoot(pk); sr != nil {
		r = c.wrapRoot(sr)
	}
	return
}

type Root struct {
	*skyobject.Root
	c *Container
}

func (r *Root) send(rp data.RootPack) {
	if !r.c.client.hasFeed(r.Pub()) {
		return // don't send
	}
	r.c.client.sendMessage(&RootMsg{
		Feed:     r.Pub(),
		RootPack: rp,
	})
}

func (r *Root) Touch() (rp data.RootPack, err error) {
	// TODO: sending never see the (*Client).feeds
	if rp, err = r.Root.Touch(); err == nil {
		r.send(rp)
	}
	return
}

func (r *Root) Inject(schemName string, i interface{}) (inj skyobject.Dynamic,
	rp data.RootPack, err error) {

	// TODO: sending never see the (*Client).feeds
	if inj, rp, err = r.Root.Inject(schemName, i); err == nil {
		r.send(rp)
	}
	return
}

func (r *Root) InjectMany(schemaName string,
	i ...interface{}) (injs []skyobject.Dynamic, rp data.RootPack,
	err error) {

	// TODO: sending never see the (*Client).feeds
	if injs, rp, err = r.Root.InjectMany(schemaName, i...); err == nil {
		r.send(rp)
	}
	return
}

func (r *Root) Replace(refs []skyobject.Dynamic) (prev []skyobject.Dynamic,
	rp data.RootPack, err error) {

	// TODO: sending never see the (*Client).feeds
	if prev, rp, err = r.Root.Replace(refs); err == nil {
		r.send(rp)
	}
	return
}

func (r *Root) Walker() (w *RootWalker) {
	return NewRootWalker(r)
}
