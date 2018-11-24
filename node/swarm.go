package node

import (
	"bytes"
	"errors"
	"fmt"
	mathrand "math/rand"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/skycoin/cxo/node/msg"
	"github.com/skycoin/skycoin/src/cipher"
)

var (
	errPeerNotFound = errors.New("peer not found")
)

// Peer
type Peer struct {
	PubKey     cipher.PubKey
	Metadata   []byte
	TCPAddr    string
	UDPAddr    string
	LastSeen   time.Time
	RetryTimes uint64
}

func msgToPeer(m msg.PeerInfo) Peer {
	p := Peer{
		PubKey:   m.PubKey,
		Metadata: m.Metadata,
		TCPAddr:  m.TCPAddr,
		UDPAddr:  m.UDPAddr,
	}

	return p
}

func peerToMsg(p Peer) msg.PeerInfo {
	m := msg.PeerInfo{
		PubKey:   p.PubKey,
		Metadata: p.Metadata,
		TCPAddr:  p.TCPAddr,
		UDPAddr:  p.UDPAddr,
	}

	return m
}

func (p *Peer) update(pi msg.PeerInfo) bool {
	needUpdate := bytes.Compare(p.Metadata, pi.Metadata) != 0 ||
		p.TCPAddr != pi.TCPAddr ||
		p.UDPAddr != pi.UDPAddr

	if needUpdate {
		p.Metadata = pi.Metadata
		p.TCPAddr = pi.TCPAddr
		p.UDPAddr = pi.UDPAddr
	}

	return needUpdate
}

func (p *Peer) seen() {
	p.LastSeen = time.Now()
}

// SwarmTracker
type SwarmTracker struct {
	cfg  SwarmTrackerConfig
	feed cipher.PubKey

	node *Node

	quit chan struct{}
	done chan struct{}

	mu sync.RWMutex

	peers map[cipher.PubKey]Peer
}

func newSwarmTracker(cfg SwarmTrackerConfig) *SwarmTracker {
	// Set default configuration if necessary
	dcfg := DefaultSwarmTrackerConfig()

	if cfg.MaxPeers == 0 {
		cfg.MaxPeers = dcfg.MaxPeers
	}
	if cfg.RequestPeerRate == 0 {
		cfg.RequestPeerRate = dcfg.RequestPeerRate
	}
	if cfg.PeerExpirePeriod == 0 {
		cfg.PeerExpirePeriod = dcfg.PeerExpirePeriod
	}
	if cfg.ClearOldPeersRate == 0 {
		cfg.ClearOldPeersRate = dcfg.ClearOldPeersRate
	}
	if cfg.MaxConns == 0 {
		cfg.MaxConns = dcfg.MaxConns
	}
	if cfg.OutgoingConnRate == 0 {
		cfg.OutgoingConnRate = dcfg.OutgoingConnRate
	}
	if cfg.PeersPerResponse == 0 {
		cfg.PeersPerResponse = dcfg.PeersPerResponse
	}

	t := &SwarmTracker{
		cfg:   cfg,
		quit:  make(chan struct{}),
		done:  make(chan struct{}),
		peers: make(map[cipher.PubKey]Peer),
	}

	return t
}

func (t *SwarmTracker) AddPeer(
	pk cipher.PubKey, meta []byte, tcp, udp string) error {

	if err := validatePeer(pk, tcp, udp); err != nil {
		return fmt.Errorf("invalid peer: %s", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if uint64(len(t.peers)) >= t.cfg.MaxPeers {
		return fmt.Errorf("maximum number of peers exceeded")
	}

	p := Peer{
		PubKey:   pk,
		Metadata: meta,
		TCPAddr:  tcp,
		UDPAddr:  udp,
		LastSeen: time.Now(),
	}
	t.peers[pk] = p

	return nil
}

func (t *SwarmTracker) RemovePeer(pk cipher.PubKey) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.peers[pk]; !ok {
		return errPeerNotFound
	}

	delete(t.peers, pk)

	return nil
}

func (t *SwarmTracker) Peers() []Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	peers := make([]Peer, 0, len(t.peers))
	for _, p := range t.peers {
		peers = append(peers, p)
	}

	return peers
}

func (t *SwarmTracker) run() {
	var (
		requestPeers  = time.Tick(t.cfg.RequestPeerRate)
		clearOldPeers = time.Tick(t.cfg.ClearOldPeersRate)
		outgoingConn  = time.Tick(t.cfg.OutgoingConnRate)
	)

LOOP:
	for {
		select {
		case <-t.quit:
			break LOOP
		default:
		}

		select {
		case <-requestPeers:
			if t.needPeers() {
				t.requestPeers()
			}

		case <-clearOldPeers:
			t.clearOldPeers()

		case <-outgoingConn:
			if count, ok := t.needConns(); ok {
				t.createOutgoingConns(count)
			}
		}
	}

	close(t.done)
}

func (t *SwarmTracker) shutdown() {
	close(t.quit)
	<-t.done
}

func (t *SwarmTracker) requestPeers() {
	var (
		conns = t.node.ConnectionsOfFeed(t.feed)
		wg    sync.WaitGroup
	)

	for i := range conns {
		wg.Add(1)

		go func(c *Conn) {
			defer wg.Done()

			// Send request
			req := &msg.RqPeers{
				Feed: t.feed,
			}
			resp, err := c.sendRequest(req)
			if err != nil {
				// TODO: log error
				// TODO: maybe call t.incPeerRetryTimes(c.PeerID())
				return
			}

			// Handle response
			switch m := resp.(type) {
			case *msg.Peers:
				if m.Feed != t.feed {
					// TODO: log error
					// TODO: maybe call t.incPeerRetryTimes(c.PeerID())
				}
				t.addPeers(m.List)

			case *msg.Err:
				// TODO: log error
				// TODO: maybe call t.incPeerRetryTimes(c.PeerID())

			default:
				// TODO: log error
				// TODO: maybe call t.incPeerRetryTimes(c.PeerID())
			}

		}(conns[i])
	}
}

func (t *SwarmTracker) needPeers() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := t.cfg.MaxPeers > 0 &&
		uint64(len(t.peers)) < t.cfg.MaxPeers

	return result
}

func (t *SwarmTracker) addPeers(peers []msg.PeerInfo) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cfg.MaxPeers > 0 && t.cfg.MaxPeers <= uint64(len(t.peers)) {
		// TODO: log warning
		return
	}

	// Validate peers
	var validPeers []msg.PeerInfo
	for _, pi := range peers {
		if err := validatePeer(
			pi.PubKey, pi.TCPAddr, pi.UDPAddr); err != nil {
			// TODO: log error
			continue
		}
		validPeers = append(validPeers, pi)
	}
	peers = validPeers

	// Shuffle and cap peers
	mathrand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	if t.cfg.MaxPeers > 0 {
		rcap := t.cfg.MaxPeers - uint64(len(t.peers))
		peers = peers[:rcap]
	}

	// Update existing peers and add new one
	for _, pi := range peers {
		p, ok := t.peers[pi.PubKey]
		if !ok {
			p.seen()
			if ok = p.update(pi); ok {
				if t.cfg.OnPeerUpdate != nil {
					go t.cfg.OnPeerUpdate(p)
				}
			}
		} else {
			p = msgToPeer(pi)
			if t.cfg.OnPeerAdded != nil {
				go t.cfg.OnPeerAdded(p)
			}
		}

		t.peers[p.PubKey] = p
	}
}

func (t *SwarmTracker) clearOldPeers() {
	now := time.Now()

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, p := range t.peers {
		if now.Sub(p.LastSeen) > t.cfg.PeerExpirePeriod {
			delete(t.peers, p.PubKey)
			if t.cfg.OnPeerRemoved != nil {
				go t.cfg.OnPeerRemoved(p)
			}
		}
	}
}

func (t *SwarmTracker) needConns() (uint64, bool) {
	var (
		connCap     = uint64(t.node.connCap())
		pendConnCap = uint64(t.node.pendingConnCap())
		feedConns   = uint64(len(t.node.ConnectionsOfFeed(t.feed)))
	)

	if connCap == 0 ||
		pendConnCap == 0 ||
		feedConns >= t.cfg.MaxConns {

		return 0, false
	}

	needFeedConns := t.cfg.MaxConns - feedConns
	if needFeedConns >= connCap {
		return connCap, true
	}

	return needFeedConns, true
}

func (t *SwarmTracker) createOutgoingConns(count uint64) {
	conns := make(map[cipher.PubKey]struct{})
	for _, c := range t.node.Connections() {
		conns[c.PeerID()] = struct{}{}
	}

	var (
		hasConn = func(p Peer) bool {
			_, ok := conns[p.PubKey]
			return !ok
		}

		peers = t.randomPeers(int(count), hasConn)

		wg sync.WaitGroup
	)

	for i := range peers {
		wg.Add(1)

		go func(p Peer) {
			defer wg.Done()

			var (
				conn *Conn
				err  error
			)

			conn, err = t.node.TCP().Connect(p.TCPAddr)
			if err != nil {
				// TODO: log error
			} else {
				if err = conn.Subscribe(t.feed); err != nil {
					// TODO: log error
				}
			}

			if err != nil {
				t.incPeerRetryTimes(conn.PeerID())
			} else {
				t.resetPeerRetryTimes(conn.PeerID())
			}
		}(peers[i])
	}
}

type peerFilter func(Peer) bool

func (t *SwarmTracker) incPeerRetryTimes(pk cipher.PubKey) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	p, ok := t.peers[pk]
	if !ok {
		return errPeerNotFound
	}

	p.RetryTimes++
	t.peers[pk] = p

	return nil
}

func (t *SwarmTracker) resetPeerRetryTimes(pk cipher.PubKey) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	p, ok := t.peers[pk]
	if !ok {
		return errPeerNotFound
	}

	p.RetryTimes--
	t.peers[pk] = p

	return nil
}

func (t *SwarmTracker) peersForExchange() []msg.PeerInfo {
	var (
		zeroRetries = func(p Peer) bool {
			return p.RetryTimes == 0
		}

		peers = t.randomPeers(int(t.cfg.PeersPerResponse), zeroRetries)

		peerMsg = make([]msg.PeerInfo, len(peers))
	)

	for i, p := range peers {
		peerMsg[i] = peerToMsg(p)
	}

	return peerMsg
}

func (t *SwarmTracker) randomPeers(count int, filters ...peerFilter) []Peer {
	filteredPeers := t.filterPeers(filters...)

	if count > len(filteredPeers) {
		count = len(filteredPeers)
	}

	var (
		peerIdx = mathrand.Perm(len(filteredPeers))[:count]
		peers   = make([]Peer, len(peerIdx))
	)
	for i, idx := range peerIdx {
		peers[i] = filteredPeers[idx]
	}

	return peers
}

func (t *SwarmTracker) filterPeers(filters ...peerFilter) []Peer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var peers []Peer

PeerLoop:
	for _, p := range t.peers {
		for _, f := range filters {
			if !f(p) {
				continue PeerLoop
			}
		}

		peers = append(peers, p)
	}

	return peers
}

func validatePeer(pk cipher.PubKey, tcp, udp string) error {
	if (pk == cipher.PubKey{}) {
		return errors.New("invalid public key")
	}
	if tcp != "" {
		if err := validateAddress(tcp); err != nil {
			return fmt.Errorf("invlaid tcp address: %s", err)
		}
	}
	if udp != "" {
		if err := validateAddress(udp); err != nil {
			return fmt.Errorf("invalid udp address: %s", err)
		}
	}

	return nil
}

func validateAddress(addr string) error {
	var (
		whitespaceFilter = regexp.MustCompile(`\s`)
		ipPort           = whitespaceFilter.ReplaceAllString(addr, "")
		split            = strings.Split(ipPort, ":")
	)
	if len(split) != 2 {
		return errors.New("invalid format")
	}
	if ip := net.ParseIP(split[0]); ip != nil {
		return errors.New("invalid ip")
	}
	if _, err := strconv.ParseUint(split[1], 10, 16); err != nil {
		return errors.New("invlaid port")
	}

	return nil
}
