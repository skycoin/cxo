package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/peterh/liner"

	"github.com/skycoin/skycoin/src/daemon/gnet"
)

const (
	// defaults
	ADDRESS = "127.0.0.1"
	PORT    = 7788
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[gnet example app] ")

	// register messages
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PING"), Ping{})
	gnet.RegisterMessage(gnet.MessagePrefixFromString("PONG"), Pong{})
	gnet.VerifyMessages()

	conf := gnet.Config{
		Address:                  ADDRESS,
		Port:                     PORT,
		MaxConnections:           1000,
		MaxMessageLength:         8192,
		DialTimeout:              5 * time.Second,
		ReadTimeout:              0,
		WriteTimeout:             0,
		EventChannelSize:         20,
		BroadcastResultSize:      20,
		ConnectionWriteQueueSize: 4096,
		DisconnectCallback: func(c *gnet.Connection,
			reason gnet.DisconnectReason) {

			log.Printf("disconnect from: %s, reason: %v", c.String(), reason)

		},
		ConnectCallback: func(c *gnet.Connection, solicited bool) {

			log.Printf("connect: %s, solicited: %t", c.String(), solicited)

		},
	}

	// get address and port from flags
	var (
		port int
	)
	flag.StringVar(&conf.Address, "a", ADDRESS, "listening address")
	flag.IntVar(&port, "p", PORT, "listening port")
	flag.Parse()
	conf.Port = uint16(port)

	//
	// create connection pool
	//
	cpool := gnet.NewConnectionPool(conf, nil)

	// start liner
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)
	line.SetCompleter(autoComplite)
	line.SetTabCompletionStyle(liner.TabPrints)

	log.Printf("listen: %s:%d", conf.Address, conf.Port)
	var err error
	if err = cpool.StartListen(); err != nil {
		log.Print("start listen error: ", err)
		return
	}
	defer cpool.StopListen()

	// accept connections from separate goroutine
	stopHandling := make(chan struct{})
	go cpool.AcceptConnections()
	go func(chan<- struct{}) {
		for {
			select {
			case <-stopHandling:
				return
			default:
			}
			cpool.HandleMessages()
		}
	}(stopHandling)
	defer close(stopHandling)

	// prompt loop
	var inpt string
	fmt.Println("enter 'help' to get help")
	for {
		inpt, err = line.Prompt("> ")
		if err != nil {
			log.Print("fatal: ", err)
			return
		}
		inpt = strings.TrimSpace(strings.ToLower(inpt))
		switch {

		case strings.HasPrefix(inpt, "connect"):
			_, err = cpool.Connect(trim(inpt, "connect"))
			if err != nil {
				log.Print("connection error: ", err)
			}

		case strings.HasPrefix(inpt, "disconnect"):
			for _, c := range cpool.GetConnections() {
				//log.Print("disconnect from: ", c)
				cpool.Disconnect(c, nil)
			}
			continue

		case strings.HasPrefix(inpt, "ping"):
			// unfortunately, we can't use Ping{}, but we can use &Ping{}
			cpool.BroadcastMessage(&Ping{Meta: trim(inpt, "ping")})

		case strings.HasPrefix(inpt, "list"):
			for _, c := range cpool.GetConnections() {
				fmt.Printf(`  %v
`, c.Conn.RemoteAddr())
			}

		case strings.HasPrefix(inpt, "info"):
			fmt.Printf(`
	address: %s:%d

`, conf.Address, conf.Port)

		case strings.HasPrefix(inpt, "help"):
			fmt.Print(`
	connect		connect to remote application using host:port
	disconnect	disconnect from remote application
	ping		send PING message with optional meta string
	list		list connections
	info		show this address
	help		show this help message
	exit		stop application
	quit

`)

		case strings.HasPrefix(inpt, "exit"):
			fallthrough

		case strings.HasPrefix(inpt, "quit"):
			fmt.Println("cya")
			return

		case inpt == "":
			continue // do noting properly

		default:
			fmt.Println("unknown command:", inpt)
			continue // no errors, no history

		}
		line.AppendHistory(inpt)
	}
}

// trim command name and spaes
func trim(inpt, cmd string) string {
	return strings.TrimSpace(strings.TrimPrefix(inpt, cmd))
}

//
// liner autocomplite
//

var complets = []string{
	"connect ",
	"disconnect ",
	"ping ",
	"list ",
	"info",
	"help",
	"exit ",
	"quit ",
}

func autoComplite(line string) (cm []string) {
	if line == "" {
		return complets
	}
	for _, c := range complets {
		if strings.HasPrefix(c, strings.ToLower(line)) {
			cm = append(cm, c)
		}
	}
	return
}

//
// Messages
//

type Ping struct {
	Meta string
}

func (p *Ping) Handle(ctx *gnet.MessageContext, _ interface{}) error {
	fmt.Println("[PING] (" + p.Meta + ")")
	fmt.Println("  connection id:            ", ctx.Conn.Id)
	fmt.Println("  connection remote address:", ctx.Conn.Conn.RemoteAddr())
	fmt.Println("----------------------------------------------")
	ctx.Conn.ConnectionPool.SendMessage(ctx.Conn, &Pong{p.Meta})
	return nil
}

type Pong struct {
	Meta string
}

func (p *Pong) Handle(ctx *gnet.MessageContext, _ interface{}) error {
	fmt.Println("[PONG] (" + p.Meta + ")")
	fmt.Println("  connection id:            ", ctx.Conn.Id)
	fmt.Println("  connection remote address:", ctx.Conn.Conn.RemoteAddr())
	fmt.Println("----------------------------------------------")
	return nil
}
