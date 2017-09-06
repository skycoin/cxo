package node

import (
	"errors"
	"net"
	"net/rpc"
	"time"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/skyobject"
)

// A RPC is receiver type for net/rpc.
// It used internally by Server and it's
// exported because net/rpc requires
// exported types. You don't need to
// use the RPC manually
type RPC struct {
	l  net.Listener
	rs *rpc.Server
	ns *Node
}

type rpcServer struct {
	*RPC
}

func newRPC(s *Node) (r *rpcServer) {
	r = new(rpcServer)
	r.RPC = new(RPC)
	r.ns = s
	r.rs = rpc.NewServer()
	return
}

func (r *rpcServer) Start(address string) (err error) {
	r.rs.RegisterName("cxo", r.RPC)
	if r.l, err = net.Listen("tcp", address); err != nil {
		return
	}
	r.ns.await.Add(1)
	go func(l net.Listener) {
		defer r.ns.await.Done()
		r.rs.Accept(l)
	}(r.l)
	return
}

func (r *rpcServer) Address() (address string) {
	if r.l != nil {
		address = r.l.Addr().String()
	}
	return
}

func (r *rpcServer) Close() (err error) {
	if r.l != nil {
		err = r.l.Close()
	}
	return
}

//
// RPC methods
//

// AddFeed to the Node
func (r *RPC) AddFeed(pk cipher.PubKey, _ *struct{}) error {
	return r.ns.AddFeed(pk)
}

// DelFeed of the node
func (r *RPC) DelFeed(pk cipher.PubKey, _ *struct{}) error {
	return r.ns.DelFeed(pk)
}

// ConnFeed used by RPC to subscribe or
// unsubscribe a connection to/from
// a feed
type ConnFeed struct {
	Address string // remote address
	Feed    cipher.PubKey
}

// Subscribe to a connection + feed
func (r *RPC) Subscribe(cf ConnFeed, _ *struct{}) (err error) {
	if cf.Address == "" {
		return errors.New("missing address")
	}
	if c := r.ns.Connection(cf.Address); c != nil {
		return c.Subscribe(cf.Feed)
	}
	return errors.New("no such connection: " + cf.Address)
}

// Unsubscribe from a connection + feed
func (r *RPC) Unsubscribe(cf ConnFeed, _ *struct{}) (_ error) {
	if cf.Address == "" {
		return errors.New("missing address")
	}
	if c := r.ns.Connection(cf.Address); c != nil {
		c.Unsubscribe(cf.Feed)
		return
	}
	return errors.New("no such connection: " + cf.Address)
}

// Feeds of a node
func (r *RPC) Feeds(_ struct{}, list *[]cipher.PubKey) (_ error) {
	*list = r.ns.Feeds()
	return
}

// A NodeConnection used by RPC and
// represents brief information about
// a connection
type NodeConnection struct {
	Address    string
	IsIncoming bool
	IsPending  bool
}

// Connections of a node
func (r *RPC) Connections(_ struct{}, list *[]NodeConnection) (_ error) {
	gcs := r.ns.Pool().Connections()
	if len(gcs) == 0 {
		return
	}

	cs := make([]NodeConnection, 0, len(gcs))

	for _, gc := range gcs {
		cs = append(cs, NodeConnection{
			Address:    gc.Address(),
			IsIncoming: gc.IsIncoming(),
			IsPending:  gc.Value() == nil,
		})
	}

	*list = cs

	return
}

// IncomingConnections of a node
func (r *RPC) IncomingConnections(_ struct{},
	list *[]NodeConnection) (_ error) {

	gcs := r.ns.Pool().Connections()
	if len(gcs) == 0 {
		return
	}

	cs := make([]NodeConnection, 0, len(gcs))

	for _, gc := range gcs {
		if false == gc.IsIncoming() {
			continue
		}
		cs = append(cs, NodeConnection{
			Address:    gc.Address(),
			IsIncoming: true,
			IsPending:  gc.Value() == nil,
		})
	}

	*list = cs
	return
}

// OutgoingConnections of a node
func (r *RPC) OutgoingConnections(_ struct{},
	list *[]NodeConnection) (_ error) {

	gcs := r.ns.Pool().Connections()
	if len(gcs) == 0 {
		return
	}

	cs := make([]NodeConnection, 0, len(gcs))

	for _, gc := range gcs {
		if true == gc.IsIncoming() {
			continue
		}
		cs = append(cs, NodeConnection{
			Address:    gc.Address(),
			IsIncoming: false,
			IsPending:  gc.Value() == nil,
		})
	}

	*list = cs
	return
}

// Connect to a remote peer. The call blocks
func (r *RPC) Connect(address string, _ *struct{}) (err error) {
	_, err = r.ns.Connect(address)
	return
}

// Disconnect from a remote peer
func (r *RPC) Disconnect(address string, _ *struct{}) (err error) {
	if c := r.ns.pool.Connection(address); c != nil {
		return c.Close()
	}
	return errors.New("no such connection: " + address)
}

// ListeningAddress of Node
func (r *RPC) ListeningAddress(_ struct{}, address *string) (_ error) {
	*address = r.ns.pool.Address()
	return
}

// A NodeInfo represents beif
// information about a node,
// excluding statistic
type NodeInfo struct {
	IsListening      bool      // is it listening
	ListeningAddress string    // listening address
	Discovery        Addresses // dicovery addresses
	IsPublicServer   bool      // is it a public server
}

// Info is RPC method that returns brief information about the Node
func (r *RPC) Info(_ struct{}, info *NodeInfo) (_ error) {
	*info = NodeInfo{
		IsListening:      r.ns.conf.EnableListener,
		ListeningAddress: r.ns.pool.Address(),
		Discovery:        r.ns.conf.DiscoveryAddresses,
		IsPublicServer:   r.ns.conf.PublicServer,
	}
	return
}

// A RootInfo used by RPC
type RootInfo struct {
	Time time.Time     // timestamp
	Seq  uint64        // seq number
	Hash cipher.SHA256 // Hash
	Prev cipher.SHA256 // hash of previous or blank if seq is 0

	CreateTime time.Time // db created at
	AccessTime time.Time // db last access
	RefsCount  uint32    // db refs count
}

func decodeRoot(val []byte) (r *skyobject.Root, err error) {
	r = new(skyobject.Root)
	if err = encoder.DeserializeRaw(val, r); err != nil {
		r = nil
	}
	return
}

// Roots returns basic information about all root obejcts of a feed.
// It returns (by RPC) list sorted from old roots to new
func (r *RPC) Roots(feed cipher.PubKey, roots *[]RootInfo) (err error) {
	ris := make([]RootInfo, 0)
	err = r.ns.DB().IdxDB().Tx(func(feeds data.Feeds) (err error) {
		var rs data.Roots
		if rs, err = feeds.Roots(feed); err != nil {
			return
		}
		return rs.Ascend(func(ir *data.Root) (err error) {
			var ri RootInfo
			ri.Hash = ir.Hash
			ri.Seq = ir.Seq

			ri.CreateTime = time.Unix(0, ir.CreateTime)
			ri.AccessTime = time.Unix(0, ir.AccessTime)

			var val []byte
			var rc uint32
			if val, rc, err = r.ns.DB().CXDS().Get(ir.Hash); err != nil {
				return
			}

			ri.RefsCount = rc

			var r *skyobject.Root
			if r, err = decodeRoot(val); err != nil {
				return
			}

			ri.Time = time.Unix(0, r.Time)
			ri.Prev = r.Prev

			ris = append(ris, ri)
			return
		})
	})
	if err != nil {
		return
	}
	*roots = ris
	return
}

// A SelectRoot used by RPC to
// select Root object
type SelectRoot struct {
	Pub      cipher.PubKey // feed
	Seq      uint64        // seq number
	LastFull bool          // ignore the seq and print last full of the feed
}

// Tree prints objects tree of chosen root object (chosen by pk+seq)
func (r *RPC) Tree(sel SelectRoot, tree *string) (err error) {
	var root *skyobject.Root
	if sel.LastFull == true {
		root, err = r.ns.so.LastRoot(sel.Pub)
	} else {
		root, err = r.ns.so.Root(sel.Pub, sel.Seq)
	}
	if err != nil {
		return
	}
	if root == nil {
		*tree = "<not found>"
	}
	*tree = r.ns.so.Inspect(root)
	return
}

// A Stat represetns Node statistic
type Stat struct {
	CXO  skyobject.Stat // skyobject stat
	Node struct {
		FillAvg     time.Duration // avg filling time of filled roots
		DropAvg     time.Duration // avg filling time of drpped roots
		Connections int           // amount of connections
		Feeds       int           // amount of feeds
	} // node stat
}

func (s *Node) feesLen() int {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	return len(s.feeds)
}

func (s *Node) connsLen() int {
	s.fmx.Lock()
	defer s.fmx.Unlock()

	return len(s.conns)
}

// Stat of database
func (r *RPC) Stat(_ struct{}, stat *Stat) (err error) {

	var st Stat
	if st.CXO, err = r.ns.so.Stat(); err != nil {
		return
	}

	st.Node.Connections = r.ns.connsLen()
	st.Node.Feeds = r.ns.feesLen()
	st.Node.FillAvg = r.ns.fillavg.Value()
	st.Node.DropAvg = r.ns.dropavg.Value()

	*stat = st
	return
}

// Terminate remote Node if allowed by its configurations
func (r *RPC) Terminate(_ struct{}, _ *struct{}) (err error) {
	if !r.ns.conf.RemoteClose {
		err = errors.New("not allowed")
		return
	}
	r.ns.Close()
	return
}
