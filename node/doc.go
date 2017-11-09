// Package node implements P2P transport for sharing CX objects.
// The node based on node/gnet and skyobject/ packages.
//
// A Node is P2P node that can connect to another nodes and can accept
// connections from another nodes
//
// A Container represents wrapper of skyobject.Container. And Root is
// wrapper of skyobject.Root. There's only one difference between
// skyobject.Root and node.Root. A node.Root sends all cahnges to
// connected peers that subscribed to feed of the Root. A Container
// returns node.Root instead of skyobject.Root
//
// Note. The node package uses Value and SetValue methods of every
// gnet.Conn for internal needs
package node
