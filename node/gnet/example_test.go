package gnet_test

import (
	"flag"
	"os"
	"os/signal"

	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/cxo/node/log"
)

type A struct {
	Value string
}

type B struct {
	Value []byte
}

type C struct {
	Value uint32
}

func Example() {

	// configs for pool and for logger

	pc, lc := gnet.NewConfig(), log.NewConfig()

	// obtain values from commandline flags

	pc.FromFlags()
	lc.FromFlags()

	flag.Parse()

	// use the logger

	pc.Logger = log.NewLogger("["+lc.Prefix+"] ", lc.Debug)

	// create pool

	pool := gnet.NewPool(pc)
	defer pool.Close()

	// register types to send/receive

	pool.Register(gnet.NewPrefix("AMSG"), &A{})
	pool.Register(gnet.NewPrefix("BMSG"), &B{})
	pool.Register(gnet.NewPrefix("CMSG"), &C{})

	// start listener if need

	if err := pool.Listen(""); err != nil {
		pool.Print(err) // use logger of the pool
		return
	}

	// connect to remote pool if need

	if err := pool.Connect("[::]:55000"); err != nil {
		pool.Print(err)
		return
	}

	// subscribe to SIGING

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	// waiting for SIGINT processing messages

	for {
		select {
		case m := <-pool.Receive():
			switch m.Value.(type) {
			case *A:
				// process A
			case *B:
				// process B
			case *C:
				// process C
				continue
			}
			// reply
			m.Conn.Send(&A{
				Value: "send reply tp the connection",
			})
			// broadcast to all connections of the pool (except this connection)
			m.Conn.Broadcast(&B{
				Value: []byte("broadcast"),
			})
			// broadcast to all
			pool.Broadcast(&C{
				Value: 3,
			})
		case <-sig:
			pool.Print("got SIGINT, exiting...")
			return
		}
	}

}
