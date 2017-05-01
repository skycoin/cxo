package gnet

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/skycoin/cxo/node/log"
)

// pool related errors
var (
	// ErrClosed represent error using closed pool
	ErrClosed = errors.New("closed pool")
	// ErrConnectionsLimit reached error
	ErrConnectionsLimit = errors.New("connections limit reached")
)

// A Pool represents connections pool
// with optional listener
type Pool struct {
	conf Config

	log.Logger

	cmx   sync.Mutex
	conns map[string]*Conn

	lmx sync.Mutex
	l   net.Listener

	sem chan struct{} // connections limit

	await sync.WaitGroup

	quito sync.Once
	quit  chan struct{}
}

// NewPool create new pool instance using
// given config. It retrns an eror if configurations
// contains invalid values
func NewPool(c Config) (p *Pool, err error) {
	if err = c.Validate(); err != nil {
		return
	}
	p = new(Pool)
	p.conf = c

	if c.Logger == nil {
		p.Logger = log.NewLogger("[pool] ", false)
	} else {
		p.Logger = c.Logger
	}

	p.conns = make(map[string]*Conn, c.MaxConnections)

	if c.MaxConnections != 0 {
		p.sem = make(chan struct{}, c.MaxConnections)
	}

	p.quit = make(chan struct{})
	return
}

// ========================================================================== //
//                                connections limit                           //
// ========================================================================== //

func (p *Pool) acquire() (got bool) {
	if got = (p.sem == nil); !got {
		select {
		case p.sem <- struct{}{}:
			got = true
		default:
		}
	}
	return
}

func (p *Pool) acquireBlock() (err error) {
	if p.sem != nil {
		select {
		case p.sem <- struct{}{}: // got
		case <-p.quit:
			err = ErrClosed
		}
	}
	return
}

func (p *Pool) release() {
	if p.sem != nil {
		<-p.sem
	}
}

// ========================================================================== //
//                                  listeninng                                //
// ========================================================================== //

func (p *Pool) Listen(address string) (err error) {
	var l net.Listener
	if p.conf.TLSConfig == nil {
		l, err = net.Listen("tcp", address)
	} else {
		l, err = tls.Listen("tcp", address, p.conf.TLSConfig)
	}
	if err != nil {
		return
	}
	p.lmx.Lock()
	defer p.lmx.Unlock()
	p.l = l
	p.await.Add(1)
	go p.listen(l)
	return
}

// accept loop
func (p *Pool) listen(l net.Listener) {
	defer p.await.Done()
	defer l.Close()

	p.Debug("start accept loop")
	defer p.Debug("stop accept loop")

	var (
		c   net.Conn
		err error

		ch ConnectionHandler
		cn *Conn
	)

	for {
		if err = p.acquireBlock(); err != nil {
			return // err closed
		}
		if c, err = l.Accept(); err != nil {
			p.release()
			select {
			case <-p.quit: // ingore the error
			default:
				p.Print("accept error: ", err)
			}
			return
		}
		if cn, err = p.acceptConnection(c); err != nil {
			p.release()
			p.Print("[ERR] accepting connection: ", err)
		} else if ch = p.conf.ConnectionHandler; ch != nil {
			ch(cn)
		}
	}
}

// Address returns listening addres or empty
// string if the Pool doesn't listen
func (p *Pool) Address() (address string) {
	p.lmx.Lock()
	defer p.lmx.Unlock()
	if p.l != nil {
		address = p.l.Addr().String()
	}
	return
}

// ========================================================================== //
//                                     dial                                   //
// ========================================================================== //

func (p *Pool) has(address string) (got bool) {
	p.cmx.Lock()
	defer p.cmx.Unlock()
	_, got = p.conns[address]
	return
}

func (p *Pool) delete(address string) {
	p.cmx.Lock()
	defer p.cmx.Unlock()
	delete(p.conns, address)
}

// Dial to remote node. The call is non-blocking it returns
// error if connection already exists, address is malformed or
// connections limit reached by the Pool
func (p *Pool) Dial(address string) (cn *Conn, err error) {
	p.cmx.Lock()
	defer p.cmx.Unlock()

	// check
	var got bool
	if _, got = p.conns[address]; got {
		err = fmt.Errorf("connection already exists %s", address)
		return
	}
	if !p.acquire() {
		err = ErrConnectionsLimit
		return
	}
	if _, err = net.ResolveTCPAddr("tcp", address); err != nil {
		return
	}

	// TODO: check out TLS config to returns immmediately if
	// the config is malformed

	cn = p.createConnection(address)
	if ch := p.conf.ConnectionHandler; ch != nil {
		ch(cn)
	}
	return
}

// ========================================================================== //
//                                connections                                 //
// ========================================================================== //

// Connections returns list of all connections
func (p *Pool) Connections() (cs []string) {
	p.cmx.Lock()
	defer p.cmx.Unlock()
	if len(p.conns) == 0 {
		return
	}
	cs = make([]string, 0, len(p.conns))
	for a := range p.conns {
		cs = append(cs, a)
	}
	return
}

// Connections returns a connection by address
// or nil if connectios with given address doesn't exists
func (p *Pool) Connection(address string) *Conn {
	p.cmx.Lock()
	defer p.cmx.Unlock()
	return p.conns[address]
}

// ========================================================================== //
//                                   close                                    //
// ========================================================================== //

func (p *Pool) closeListener() (err error) {
	p.lmx.Lock()
	defer p.lmx.Unlock()
	if p.l != nil {
		err = p.l.Close()
	}
	return
}

func (p *Pool) closeConnections() {
	p.cmx.Lock()
	defer p.cmx.Unlock()
	for _, c := range p.conns {
		p.cmx.Unlock()
		c.Close()
		p.cmx.Lock()
	}
}

// Close the Pool
func (p *Pool) Close() (err error) {
	p.quito.Do(func() {
		close(p.quit)
		err = p.closeListener()
		p.closeConnections()
		p.await.Wait()
	})
	return
}
