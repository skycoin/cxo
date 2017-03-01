package node

import (
	"fmt"
	"sync"
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

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
		mcx.Remote().PubKey(),
		mcx.Remote().Address()}
	rp.Local = RouteSide{
		mcx.Local().PubKey(),
		mcx.Local().Address()}
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
		r.Local.Address, r.Local.Pub.Hex(),
		r.Remote.Address, r.Remote.Pub.Hex())
}

// SendReciveStore keeps routes from inside itself.
// It designed to use with SendReceive messages
type SendReciveStore struct {
	sync.Mutex
	LastId int64
	// id-> []{point, connection_side, route_point}
	Store map[int64][]SendReceivePoint
}

func NewSendReciveStore() *SendReciveStore {
	return &SendReciveStore{
		LastId: 0,
		Store:  make(map[int64][]SendReceivePoint),
	}
}

func (s *SendReciveStore) New() (x *SendReceive) {
	s.Lock()
	defer s.Unlock()
	s.LastId++
	x = new(SendReceive)
	x.Id = s.LastId // x.Point = 0
	return
}

func (s *SendReciveStore) Save(x *SendReceive, srp SendReceivePoint) {
	s.Lock()
	defer s.Unlock()
	stored := s.Store[x.Id]
	stored = append(stored, srp)
	s.Store[x.Id] = stored
}

func (s *SendReciveStore) Get(id int64) (x []SendReceivePoint, ok bool) {
	s.Lock()
	defer s.Unlock()
	x, ok = s.Store[id]
	return
}

// clean up to free memory
func (s *SendReciveStore) Reset() {
	s.Lock()
	defer s.Unlock()
	s.LastId = 0
	s.Store = make(map[int64][]SendReceivePoint)
}

// keep information about send-recive handling place
type SendReceivePoint struct {
	Point      int64
	Outgoing   bool // connection side
	RoutePoint RoutePoint
}

// SendReceive is message that (1) increments its Point value,
// (2) store SendRecivePoint in related store (3) recive back once
type SendReceive struct {
	Id    int64
	Point int64 // point incremented each handling
}

// handled from outgoing connection (by subscriber)
func (s *SendReceive) HandleOutgoing(oc MsgContext,
	user interface{}) (terminate error) {

	sr := user.(*SendReciveStore)

	s.Point++
	srp := SendReceivePoint{
		Point:      s.Point,
		Outgoing:   true,
		RoutePoint: CreateRoutePoint(oc)}
	sr.Save(s, srp)
	srp.RoutePoint.VerbosePrint("SendRecive", s.Id)
	// send reply
	return oc.Reply(s)
}

// handled from incoming connection (by feed)
func (s *SendReceive) HandleIncoming(ic MsgContext,
	user interface{}) (terminate error) {

	sr := user.(*SendReciveStore)

	s.Point++
	srp := SendReceivePoint{
		Point:      s.Point,
		Outgoing:   false,
		RoutePoint: CreateRoutePoint(ic)}
	sr.Save(s, srp)
	srp.RoutePoint.VerbosePrint("SendRecive", s.Id)
	// dont reply
	return
}

//
// check result (see route points one by one)
//

type ExpectedPoint struct {
	Local    cipher.PubKey // expected public key of local point
	Remote   cipher.PubKey // expected public key of remote point
	Outgoing bool          // expected handler (feed or subscriber)
}

func (sr *SendReciveStore) Lookup(id int64, ep []ExpectedPoint, t *testing.T) {
	result, ok := sr.Get(id)
	if !ok {
		t.Error("message hasn't traveled")
		return
	}

	switch lr := len(result); {
	case lr < len(ep):
		t.Error("to few message handlings: ", lr)
		return
	case lr > len(ep):
		t.Error("too many message handlings:", lr)
		return
	default: // ok
	}

	// points order
	for i, srp := range result {
		if srp.Point != int64(i+1) {
			t.Error("invalid handling order")
			return
		}
	}

	// range over expected points
	for i, e := range ep {
		srp := result[i]
		if e.Outgoing != srp.Outgoing {
			// check side
			t.Errorf("point %d: invalid handler: %d",
				i,
				sideString(srp.Outgoing))
		} else {
			// check publick keys
			if srp.RoutePoint.Local.Pub != e.Local {
				t.Error("invalid public key of local point")
			}
			if srp.RoutePoint.Remote.Pub != e.Remote {
				t.Error("invalid public key of remote point")
			}
		}
	}

}

func sideString(outgoing bool) string {
	if outgoing {
		return "subscriber"
	}
	return "feed"
}
