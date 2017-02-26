package node

import (
	"fmt"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

//
// TODO: async access for broadcasting without deadlocks
//

//
// go forward from node to node, broadcast itself
// on each node it handled
//

var forwardStore ForwardStore = ForwardStore{
	Store:  make(map[int64]Forward),
	LastId: 0,
}

type RouteSide struct {
	Pub     cipher.PubKey
	Address string
}

type RoutePoint struct {
	Remote RouteSide
	Local  RouteSide
}

func CreateRoutePoint(mcx MsgContext) (rp RoutePoint) {
	rp.Remote = RouteSide{
		ic.Remote().PubKey(),
		ic.Remote().Address()}
	rp.Local = RouteSide{
		ic.Local().PubKey(),
		ic.Local().Address()}
	return
}

func (r *RoutePoint) VerbosePrint(name string, id int64) {
	if !testing.Verbose() {
		return
	}
	fmt.Printf(`---
	%-8s    %d
	Local:  %s, %s
	Remote: %s, %s
---
`,
		name, id,
		point.Local.Address, point.Local.Pub.Hex(),
		point.Remote.Address, point.Remote.Pub.Hex())
}

// broadcast itself on each node it handled
type Forward struct {
	Id    int64
	Route []RoutePoint
}

// keep last router instance
type ForwardStore struct {
	Store  map[int64]Forward
	LastId int64
}

func (f *ForwardStore) Save(x *Forward) {
	f.Store[x.Id] = *x
}

func (f *ForwardStore) New() (x *Forward) {
	f.LastId++
	x = new(Forward)
	x.Id = f.LastId
	f.Save(x)
	return
}

func (f *ForwardStore) Get(id int64) (x Forward, ok bool) {
	x, ok = f.Store[id]
	return
}

func (f *Forward) HandleIncoming(ic IncomingContext) (terminate error) {
	point := CreateRoutePoint(ic)
	point.VerbosePrint("Forward", f.Id)
	f.Route = append(f.Route, point)
	forwardStore.Save(f)
	return ic.Broadcast(f) // broadcast itself
}

//
// send-recive (a'la ping-pong)
//

var sendReceiveStore SendReciveStore = SendReciveStore{
	LastId: 0,
	Store:  make(map[int64][]SendReceivePoint),
}

func (s *SendReciveStore) New() (x *SendReceive) {
	s.LastId++
	x = new(SendReceive)
	x.Id = s.LastId // x.Point = 0
	return
}

func (s *SendReciveStore) Save(x *SendReceive, srp SendReceivePoint) {
	stored, ok := s.Store[x.Id]
	if !ok {
		panic("can't find SendReceive point")
	}
	stored = append(stored, srp)
	s.Store[x.Id] = stored
}

func (s *SendReciveStore) Get(id int64) (x SendReceivePoint, ok bool) {
	x, ok = s.Store[id]
	return
}

// SendReciveStore opposite to ForwardStore keeps routes from inside
// the store (not message itself)
type SendReciveStore struct {
	LastId int64
	// id-> []{point, connection_side, route_point}
	Store map[int][]SendReceivePoint
}

// keep information about send-recive handling place
type SendReceivePoint struct {
	Point      int64
	Outgoing   bool // connection side
	RoutePoint RoutePoint
}

type SendReceive struct {
	Id    int64
	Point int64 // point incremented each handling
}

func (s *SendReceive) HandleOutgoing(oc OutgoingContext) (terminate error) {
	s.Point++
	srp := SendReceivePoint{
		Point:      s.Point,
		Outgoing:   true,
		RoutePoint: CreateRoutePoint(oc)}
	sendReceiveStore.Save(s, srp)
	srp.RoutePoint.VerbosePrint("SendRecive", s.Id)
	// send reply
	return oc.Reply(s)
}

// handled from incoming connection (by feed)
func (s *SendReceive) HandleIncoming(ic IncomingContext) (terminate error) {
	s.Point++
	srp := SendReceivePoint{
		Point:      s.Point,
		Outgoing:   false,
		RoutePoint: CreateRoutePoint(ic)}
	sendReceiveStore.Save(s, srp)
	srp.RoutePoint.VerbosePrint("SendRecive", s.Id)
	// dont reply
	return
}
