package node

import (
	"sync"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/net/factory"
	discovery "github.com/skycoin/net/skycoin-messenger/factory"
)

//
// TODO (kostyarin): DRY
//

// A TCP represents TCP transport
// of the Node. The TCP used to
// listen and connect
type TCP struct {
	*factory.TCPFactory // underlying transport

	// back reference
	n *Node

	//
	mx sync.Mutex

	d *discovery.MessengerFactory // underlying discovery server or nil

	address     string // listening address
	isListening bool

	cs map[string]*Conn // address -> conn
}

func newTCP(n *Node) (t *TCP) {

	t = new(TCP)

	t.TCPFactory = factory.NewTCPFactory()

	t.n = n
	t.AcceptedCallback = t.acceptConn
	t.cs = make(map[string]*Conn)

	return
}

func (t *TCP) addConn(c *Conn) {
	t.mx.Lock()
	defer t.mx.Unlock()

	t.cs[c.Address()] = c
}

func (t *TCP) delConn(c *Conn) {
	t.mx.Lock()
	defer t.mx.Unlock()

	delete(t.cs, c.Address())
}

func (t *TCP) getConn(address string) (c *Conn) {
	t.mx.Lock()
	defer t.mx.Unlock()

	return t.cs[address]
}

// Listen on given address. It's possible to listen
// only once
func (t *TCP) Listen(address string) (err error) {

	t.mx.Lock()
	defer t.mx.Unlock()

	if t.isListening == true {
		return ErrAlreadyListen
	}

	if err = t.TCPFactory.Listen(address); err != nil {
		return
	}

	t.isListening = true
	t.address = address
	return
}

// Address returns istening address as it
// passed to the Listen method. The address
// is blank string if the TCP is not listening
func (t *TCP) Address() string {
	t.mx.Lock()
	defer t.mx.Unlock()

	return t.address
}

// Connect to given TCP address. The method blocks. If connection
// with given address already exists, then the Connect returns this
// existing connection.
func (t *TCP) Connect(address string) (*Conn, error) {
	// Check if connections to/from address already exists.
	c := t.getConn(address)
	if c != nil {
		return c, nil
	}

	// Open new factory.Connection.
	fc, err := t.TCPFactory.Connect(address)
	if err != nil {
		return nil, err
	}

	// Init outgoing connection (handhshake, check duplicate pubkey, etc.)
	// In case of success connections to/from the the same address has to be blocked.
	// In case of failure incoming connection has to be closed.
	if c, err = t.n.initConn(fc, false); err == nil {
		t.addConn(c)
	} else {
		t.n.Debugf(CloseConnPin, "[%] closing factory.Connection")
		fc.Close()
	}

	return c, err
}

func (t *TCP) acceptConn(fc *factory.Connection) {
	var (
		addr = fc.GetRemoteAddr().String()
		err  error
	)

	t.n.Debugf(NewInConnPin, "[%s] accepting", connString(true, fc.IsTCP(), addr))

	// Check if connections to/from address already exists.
	c := t.getConn(addr)
	if c != nil {
		err = ErrAlreadyHaveConnection
	} else {
		// Init incoming connection (handhshake, check duplicate pubkey, etc.)
		c, err = t.n.initConn(fc, true)
	}

	// In case of success connections to/from the the same address has to be blocked.
	// In case of failure incoming connection has to be closed.
	if err == nil {
		t.addConn(c)
	} else {
		t.n.Errorf(err, "[%s] failed to accept")
		t.n.Debugf(CloseConnPin, "[%] closing factory.Connection")
		fc.Close()
	}
}

// Discovery returns underlying MessengerFactory that
// can be nil, if feature disabled
func (t *TCP) Discovery() (d *discovery.MessengerFactory) {
	t.mx.Lock()
	defer t.mx.Unlock()

	return t.d
}

// create or get related *discovery.MessengerFactory instance
func (t *TCP) discovery() (d *discovery.MessengerFactory) {
	t.mx.Lock()
	defer t.mx.Unlock()

	if t.d == nil {

		var d *discovery.MessengerFactory

		d = discovery.NewMessengerFactory()

		if t.n.config.Logger.Debug == true {
			if t.n.config.Logger.Pins&DiscoveryPin != 0 {
				d.SetLoggerLevel(discovery.DebugLevel)
			} else {
				d.SetLoggerLevel(discovery.ErrorLevel) // change to panic?
			}
		}

		t.d = d

	}

	return t.d
}

// ConnectToDiscoveryServer connects to given discovery server.
// If Discovery is nil, then it will be created
func (t *TCP) ConnectToDiscoveryServer(address string) (err error) {

	err = t.discovery().ConnectWithConfig(address, &discovery.ConnConfig{
		SeedConfig: t.n.id,

		Reconnect:     true,
		ReconnectWait: time.Second * 30,

		FindServiceNodesByKeysCallback: t.findServiceNodes,
		OnConnected:                    t.onDiscoveryConnected,
	})

	return
}

//
// discovery callbacks
//

func (t *TCP) findServiceNodes(resp *discovery.QueryResp) {

	t.n.Debug(DiscoveryPin, "(TCP) findServiceNodes:", len(resp.Result))

	for _, si := range resp.Result {

		if si == nil {
			continue // sometimes it happens, TODO (kostyarin): ask about it
		}

		for _, ni := range si.Nodes {

			if ni == nil {
				continue // never happens
			}

			var (
				c, yep = t.n.hasPeer(ni.PubKey)
				err    error
			)

			if yep == false {
				if c, err = t.Connect(ni.Address); err != nil { // block
					t.n.Debugf(DiscoveryPin, "can't Connect to tcp://%q: %v",
						ni.Address,
						err)
					continue
				}
			}

			// block
			if err = c.Subscribe(si.PubKey); err != nil {
				t.n.Debugf(DiscoveryPin, "[%s] can't Subscribe to %s: %v",
					c.Address(),
					si.PubKey.Hex()[:7],
					err)
			}

			// continue

		}

	}

}

// called after Share/DontShare
func (t *TCP) updateServiceDiscovery(feeds []cipher.PubKey) {

	if t.Discovery() == nil {
		return
	}

	t.n.Debug(DiscoveryPin, "updateServiceDiscovery")

	var services = make([]*discovery.Service, 0, len(feeds))

	for _, pk := range feeds {
		services = append(services, &discovery.Service{Key: pk})
	}

	var (
		share   = t.n.config.Public
		address string
	)

	if share == true {
		address = t.Address()
		share = share && (address != "")
	}

	t.d.ForEachConn(func(c *discovery.Connection) {

		if err := c.FindServiceNodesByKeys(feeds); err != nil {
			t.n.Debug(DiscoveryPin,
				"(TCP) (discovery.Connection).FindServiceNodesByKeys error:",
				err)
			return
		}

		if share == false {
			return
		}

		err := c.UpdateServices(&discovery.NodeServices{
			ServiceAddress: address,
			Services:       services,
		})

		if err != nil {
			t.n.Debug(DiscoveryPin,
				"(TCP) (discovery.Connection).UpdateServices error:",
				err)
		}
	})

}

func (t *TCP) onDiscoveryConnected(c *discovery.Connection) {

	t.n.Debugln(DiscoveryPin, "(UDP) OnDiscoveryConencted")
	t.updateServiceDiscovery(t.n.Feeds())

}

// connections strings
func (t *TCP) connections() (cs []string) {
	t.mx.Lock()
	defer t.mx.Unlock()

	cs = make([]string, 0, len(t.cs))

	for _, c := range t.cs {
		cs = append(cs, c.String())
	}

	return
}

// A UDP represents UDP transport
// of the Node. The UDP used to
// listen and connect
type UDP struct {
	*factory.UDPFactory // underlying transport

	// back reference
	n *Node

	mx sync.Mutex

	d *discovery.MessengerFactory

	address     string
	isListening bool

	cs map[string]*Conn // connections
}

func newUDP(n *Node) (u *UDP) {

	u = new(UDP)

	u.UDPFactory = factory.NewUDPFactory()

	u.n = n
	u.AcceptedCallback = u.acceptConn
	u.cs = make(map[string]*Conn)

	return
}

func (u *UDP) addConn(c *Conn) {
	u.mx.Lock()
	defer u.mx.Unlock()

	u.cs[c.Address()] = c
}

func (u *UDP) delConn(c *Conn) {
	u.mx.Lock()
	defer u.mx.Unlock()

	delete(u.cs, c.Address())
}

func (u *UDP) getConn(address string) (c *Conn) {
	u.mx.Lock()
	defer u.mx.Unlock()

	return u.cs[address]
}

// Listen on given address. It's possible to listen
// only once
func (u *UDP) Listen(address string) (err error) {

	u.mx.Lock()
	defer u.mx.Unlock()

	if u.isListening == true {
		return ErrAlreadyListen
	}

	if err = u.UDPFactory.Listen(address); err != nil {
		return
	}

	u.address = address
	u.isListening = true
	return
}

// Address returns istening address as it
// passed to the Listen method. The address
// is blank string if the UDP is not listening
func (u *UDP) Address() string {
	u.mx.Lock()
	defer u.mx.Unlock()

	return u.address
}

func (u *UDP) addAcceptedConnection(c *Conn) {
	u.mx.Lock()
	defer u.mx.Unlock()

	u.cs[c.Address()] = c
}

// Connect to given UDP address. If connection with given
// address already exists, then the Connect returns this
// existing connection.
func (u *UDP) Connect(address string) (*Conn, error) {
	// Check if connections to/from address already exists.
	c := u.getConn(address)
	if c != nil {
		return c, nil
	}

	// Open new factory.Connection.
	fc, err := u.UDPFactory.Connect(address)
	if err != nil {
		return nil, err
	}

	// Init outgoing connection (handhshake, check duplicate pubkey, etc.)
	// In case of success connections to/from the the same address has to be blocked.
	// In case of failure incoming connection has to be closed.
	if c, err = u.n.initConn(fc, false); err == nil {
		u.addConn(c)
	} else {
		u.n.Debugf(CloseConnPin, "[%] closing factory.Connection")
		fc.Close()
	}

	return c, err
}

func (u *UDP) acceptConn(fc *factory.Connection) {
	var (
		addr = fc.GetRemoteAddr().String()
		err  error
	)

	u.n.Debugf(NewInConnPin, "[%s] accepting", connString(true, fc.IsTCP(), addr))

	// Check if connections to/from address already exists.
	c := u.getConn(addr)
	if c != nil {
		err = ErrAlreadyHaveConnection
	} else {
		// Init incoming connection (handhshake, check duplicate pubkey, etc.)
		c, err = u.n.initConn(fc, true)
	}

	// In case of success connections to/from the the same address has to be blocked.
	// In case of failure incoming connection has to be closed.
	if err == nil {
		u.addConn(c)
	} else {
		u.n.Errorf(err, "[%s] failed to accept")
		u.n.Debugf(CloseConnPin, "[%] closing factory.Connection")
		fc.Close()
	}
}

// Discovery returns underlying MessengerFactory that
// can be nil, if feature disabled
func (u *UDP) Discovery() (d *discovery.MessengerFactory) {
	u.mx.Lock()
	defer u.mx.Unlock()

	return u.d
}

// create or get related *discovery.MessengerFactory instance
func (u *UDP) discovery() (d *discovery.MessengerFactory) {
	u.mx.Lock()
	defer u.mx.Unlock()

	if u.d == nil {

		var d *discovery.MessengerFactory

		d = discovery.NewMessengerFactory()

		if u.n.config.Logger.Debug == true {
			if u.n.config.Logger.Pins&DiscoveryPin != 0 {
				d.SetLoggerLevel(discovery.DebugLevel)
			} else {
				d.SetLoggerLevel(discovery.ErrorLevel) // change to panic?
			}
		}

		u.d = d

	}

	return u.d
}

// ConnectToDiscoveryServer connects to given discovery server.
// If Discovery is nil, then it will be created
func (u *UDP) ConnectToDiscoveryServer(address string) (err error) {

	err = u.discovery().ConnectWithConfig(address, &discovery.ConnConfig{
		SeedConfig: u.n.id,

		Reconnect:     true,
		ReconnectWait: time.Second * 30,

		FindServiceNodesByKeysCallback: u.findServiceNodes,
		OnConnected:                    u.onDiscoveryConnected,
	})

	return
}

//
// discovery callbacks
//

func (u *UDP) findServiceNodes(resp *discovery.QueryResp) {

	u.n.Debug(DiscoveryPin, "(UDP) findServiceNodes:", len(resp.Result))

	for _, si := range resp.Result {

		if si == nil {
			continue // sometimes it happens, TODO (kostyarin): ask about it
		}

		for _, ni := range si.Nodes {

			if ni == nil {
				continue // never happens
			}

			var (
				c, yep = u.n.hasPeer(ni.PubKey)
				err    error
			)

			if yep == false {
				if c, err = u.Connect(ni.Address); err != nil { // block
					u.n.Debugf(DiscoveryPin, "can't Connect to udp://%q: %v",
						ni.Address,
						err)
					continue
				}
			}

			// block
			if err = c.Subscribe(si.PubKey); err != nil {
				u.n.Debugln(DiscoveryPin, "[%s] can't Subscribe to %s: %v",
					c.Address(),
					si.PubKey.Hex()[:7],
					err)
			}

			// continue

		}

	}

}

// called after Share/DontShare
func (u *UDP) updateServiceDiscovery(feeds []cipher.PubKey) {

	if u.Discovery() == nil {
		return
	}

	u.n.Debug(DiscoveryPin, "updateServiceDiscovery")

	var services = make([]*discovery.Service, 0, len(feeds))

	for _, pk := range feeds {
		services = append(services, &discovery.Service{Key: pk})
	}

	var (
		share   = u.n.config.Public
		address string
	)

	if share == true {
		address = u.Address()
		share = share && (address != "") // skip, if it is not listening
	}

	u.d.ForEachConn(func(c *discovery.Connection) {

		if err := c.FindServiceNodesByKeys(feeds); err != nil {
			u.n.Debug(DiscoveryPin,
				"(TCP) (discovery.Connection).FindServiceNodesByKeys error:",
				err)
			return
		}

		if share == false {
			return
		}

		err := c.UpdateServices(&discovery.NodeServices{
			ServiceAddress: address,
			Services:       services,
		})

		if err != nil {
			u.n.Debug(DiscoveryPin,
				"(TCP) (discovery.Connection).UpdateServices error:",
				err)
		}
	})

}

func (u *UDP) onDiscoveryConnected(c *discovery.Connection) {

	u.n.Debugln(DiscoveryPin, "(UDP) OnDiscoveryConencted")
	u.updateServiceDiscovery(u.n.Feeds())

}

// connections strings
func (u *UDP) connections() (cs []string) {
	u.mx.Lock()
	defer u.mx.Unlock()

	cs = make([]string, 0, len(u.cs))

	for _, c := range u.cs {
		cs = append(cs, c.String())
	}

	return
}
