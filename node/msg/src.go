package msg

import (
	"crypto/rand"
	"encoding/binary"
	"errors"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

// common errors
var (
	ErrNotRequestMsg  = errors.New("not request msg")
	ErrNotResponseMsg = errors.New("nto response msg")
)

// ID represents uint64 random number
// represented as [8]byte
type ID [8]byte

// IsZero retursn true if the ID is blank
func (i ID) IsZero() bool {
	return i == (ID{})
}

// Uint64 of the ID
func (i ID) Uint64() uint64 {
	return binary.LittleEndian.Uint64(i[:])
}

// Set given value to the ID
func (i *ID) Set(u uint64) {
	binary.LittleEndian.PutUint64((*i)[:], u)
}

// An Src represents msg source that
// creates messages with unique ID
type Src struct{}

// ID retusn new randon ID
func (s *Src) ID() (id ID) {
	if _, err := rand.Read(id[:]); err != nil {
		panic("[CRIT] can'st crypto/rand.Read, fatality!")
	}
	return
}

func (s *Src) Ping() (ping *Ping) {
	ping = new(Ping)
	ping.ID = s.ID()
	return
}

func (s *Src) Pong(ping *Ping) (pong *Pong) {
	pong = new(Pong)
	pong.ResponseFor = ping.ID
	return
}

func (s *Src) Hello() (hello *Hello) {
	hello = new(Hello)
	hello.ID = s.ID()
	hello.Protocol = Version
	// hello.Address (reserved)
	return
}

func (s *Src) Accept(hello *Hello) (accept *Accept) {
	accept = new(Accept)
	accept.ResponseFor = hello.ID
	// accept.Address (reserved)
	return
}

func (s *Src) Reject(hello *Hello, err string) (reject *Reject) {
	reject = new(Reject)
	reject.ResponseFor = hello.ID
	reject.Err = err
	// reject.Address (reserved)
	return
}

func (s *Src) Subscribe(pk cipher.PubKey) (sub *Subscribe) {
	sub = new(Subscribe)
	sub.ID = s.ID()
	sub.Feed = pk
	return
}

func (s *Src) Unsubscribe(pk cipher.PubKey) (unsub *Unsubscribe) {
	unsub = new(Unsubscribe)
	unsub.ID = s.ID()
	unsub.Feed = pk
	return
}

func (s *Src) AcceptSubscription(sub *Subscribe) (as *AcceptSubscription) {
	as = new(AcceptSubscription)
	as.Feed = sub.Feed
	as.ResponseFor = sub.ID
	return
}

func (s *Src) RejectSubscription(sub *Subscribe) (as *RejectSubscription) {
	as = new(RejectSubscription)
	as.Feed = sub.Feed
	as.ResponseFor = sub.ID
	return
}

func (s *Src) RequestListOfFeeds() (rls *RequestListOfFeeds) {
	rls = new(RequestListOfFeeds)
	rls.ID = s.ID()
	return
}

func (s *Src) ListOfFeeds(rls *RequestListOfFeeds,
	list []cipher.PubKey) (lf *ListOfFeeds) {

	lf = new(ListOfFeeds)
	lf.ResponseFor = rls.ID
	lf.List = list
	return
}

func (s *Src) NonPublicServer(rls *RequestListOfFeeds) (nps *NonPublicServer) {
	nps = new(NonPublicServer)
	nps.ResponseFor = rls.ID
	return
}

func (s *Src) Root(pk cipher.PubKey, r *skyobject.Root) (root *Root) {
	root = new(Root)
	root.Feed = pk
	root.Value = r.Encode()
	root.Sig = r.Sig
	return
}

func (s *Src) RequestObject(key cipher.SHA256) (ro *RequestObject) {
	ro = new(RequestObject)
	ro.Key = key
	return
}

func (s *Src) Object(key cipher.SHA256, val []byte) (o *Object) {
	o = new(Object)
	o.Value = val
	o.Key = key
	return
}

func (s *Src) NotFound(key cipher.SHA256) (nf *NotFound) {
	nf = new(NotFound)
	nf.Key = key
	return
}
