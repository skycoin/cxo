package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterh/liner"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/rpc/client"
	"github.com/skycoin/cxo/rpc/comm"
)

var (
	ErrUnknowCommand    = errors.New("unknown command")
	ErrMisisngArgument  = errors.New("missing argument")
	ErrTooManyArguments = errors.New("too many arguments")

	commands = []string{
		"list",
		"connect",
		"subscribe",
		"disconnect",
		"inject",
		"want",
		"got",
		"info",
		"stat",
		"terminate",
		"quit",
		"exit",
	}
)

func main() {
	var (
		address string
		execute string

		rpc *client.Client
		err error

		line      *liner.State
		cmd       string
		terminate bool

		help bool
		code int
	)

	defer func() { os.Exit(code) }()

	flag.StringVar(&address,
		"a",
		"",
		"rpc address")
	flag.StringVar(&execute,
		"e",
		"",
		"execute commant and exit")

	flag.BoolVar(&help,
		"h",
		false,
		"show help")

	flag.Parse()

	if help {
		fmt.Printf("Usage %s <flags>\n", os.Args[0])
		flag.PrintDefaults()
		return
	}

	if address == "" {
		fmt.Fprintln(os.Stderr, "empty address")
		code = 1
		return
	}

	if rpc, err = client.Dial("tcp", address); err != nil {
		fmt.Fprintln(os.Stderr, "error creating rpc-clinet:", err)
		code = 1
		return
	}
	defer rpc.Close()

	if execute != "" {
		_, err = executeCommand(execute, rpc, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
		return
	}

	line = liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range commands {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	// rpompt loop

	fmt.Println("enter 'help' to get help")
	for {
		cmd, err = line.Prompt("> ")
		if err != nil && err != liner.ErrPromptAborted {
			fmt.Fprintln(os.Stderr, "fatal error:", err)
			code = 1
			return
		}
		terminate, err = executeCommand(cmd, rpc, line)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if terminate {
			return
		}
		line.AppendHistory(cmd)
	}

}

func args(ss []string) (string, error) {
	switch len(ss) {
	case 0, 1:
		return "", ErrMisisngArgument
	case 2:
		return ss[1], nil
	}
	return "", ErrTooManyArguments
}

func executeCommand(command string, rpc *client.Client,
	line *liner.State) (terminate bool, err error) {

	ss := strings.Fields(command)
	if len(ss) == 0 {
		return
	}
	switch ss[0] {
	case "list":
		err = list(rpc)
	case "connect":
		err = connect(rpc, ss)
	case "subscribe":
		err = subscribe(rpc, ss)
	case "disconnect":
		err = disconnect(rpc, ss)
	case "inject":
		err = inject(rpc, ss)
	case "want":
		err = want(rpc, ss)
	case "got":
		err = got(rpc, ss)
	case "info":
		err = info(rpc)
	case "stat":
		err = stat(rpc)
	case "terminate":
		err = term(rpc)
	case "help":
		showHelp()
	case "quit":
		fallthrough
	case "exit":
		terminate = true
		fmt.Println("cya")
		return
	default:
		err = ErrUnknowCommand
		return
	}
	return
}

func showHelp() {
	fmt.Println(`

  list
    list connections
  connect <address>
    add connection to given address
  subscribe <public key>
    subscribe to feed
  disconnect  <address>
    disconnect from given address
  inject <hash> <public key> <secret key>
    inject given hash to references of the root, update and share the root
  want <public key>
  	want returns list of hashes of missing object of given root object
  got <public key>
  	got returns list of objects given root object already has with size
  info
    obtain information about the node (listening address and list of feeds)
  stat
    obtain database statistic
  terminate
    close the node
  help
    show this help message
  quit or exit
    leave the cli

`)
}

func list(rpc *client.Client) (err error) {
	var list []string
	if list, err = rpc.List(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Println("  there aren't connections")
		return
	}
	for _, c := range list {
		fmt.Println("  -", c)
	}
	return
}

func info(rpc *client.Client) (err error) {
	var info comm.Info
	if info, err = rpc.Info(); err != nil {
		return
	}
	fmt.Println("  Listening address:", info.Address)
	if len(info.Feeds) == 0 {
		fmt.Println("  There aren't subscriptions")
		return
	}
	for _, pk := range info.Feeds {
		fmt.Println("  + ", pk.Hex())
	}
	return
}

func connect(rpc *client.Client, ss []string) (err error) {
	var address string
	if address, err = args(ss); err != nil {
		return
	}
	if err = rpc.Connect(address); err != nil {
		return
	}
	fmt.Println("  connected")
	return
}

func subscribe(rpc *client.Client, ss []string) (err error) {
	var hex string
	if hex, err = args(ss); err != nil {
		return
	}
	var pub cipher.PubKey
	if pub, err = cipher.PubKeyFromHex(hex); err != nil {
		return
	}
	if err = rpc.Subscribe(pub); err != nil {
		return
	}
	fmt.Println("  subscribed")
	return
}

func disconnect(rpc *client.Client, ss []string) (err error) {
	var address string
	if address, err = args(ss); err != nil {
		return
	}
	if err = rpc.Disconnect(address); err != nil {
		return
	}
	fmt.Println("  disconnected")
	return
}

func inject(rpc *client.Client, ss []string) (err error) {
	var hash, pub, sec string
	switch len(ss) {
	case 0, 1:
		err = errors.New("missing arguments")
		return
	case 2:
		err = errors.New("missing public key argument")
		return
	case 3:
		err = errors.New("missing secret key argument")
		return
	case 4:
		hash = ss[1]
		pub = ss[2]
		sec = ss[3]
	default:
		err = ErrTooManyArguments
		return
	}
	var args comm.Inject
	if args.Hash, err = cipher.SHA256FromHex(hash); err != nil {
		return
	}
	if args.Pub, err = cipher.PubKeyFromHex(pub); err != nil {
		return
	}
	if args.Sec, err = cipher.SecKeyFromHex(sec); err != nil {
		return
	}
	if err = rpc.Inject(args); err == nil {
		fmt.Println("  injected")
	}
	return
}

func want(rpc *client.Client, ss []string) (err error) {
	var pub string
	if pub, err = args(ss); err != nil {
		return
	}
	var public cipher.PubKey
	if public, err = cipher.PubKeyFromHex(pub); err != nil {
		return
	}
	var list []cipher.SHA256
	if list, err = rpc.Want(public); err == nil {
		if len(list) == 0 {
			fmt.Println("  no objects wanted")
			return
		}
		for _, k := range list {
			fmt.Println("  +", k.Hex())
		}
	}
	return
}

func got(rpc *client.Client, ss []string) (err error) {
	var pub string
	if pub, err = args(ss); err != nil {
		return
	}
	var public cipher.PubKey
	if public, err = cipher.PubKeyFromHex(pub); err != nil {
		return
	}
	var list map[cipher.SHA256]int
	if list, err = rpc.Got(public); err == nil {
		if len(list) == 0 {
			fmt.Println("  no objects has got")
			return
		}
		var total int
		for k, l := range list {
			total += l
			fmt.Println("  +", k.Hex(), l)
		}
		fmt.Println("  -------------------------------")
		fmt.Printf("  total objects: %d, total size %s\n",
			len(list), data.HumanMemory(total))
	}
	return
}

func stat(rpc *client.Client) (err error) {
	var stat data.Stat
	if stat, err = rpc.Stat(); err != nil {
		return
	}
	fmt.Println("  Total objects:", stat.Total)
	fmt.Println("  Memory:       ", data.HumanMemory(stat.Memory))
	return
}

func term(rpc *client.Client) (err error) {
	if err = rpc.Terminate(); err == nil {
		fmt.Println("  terminated")
	}
	return
}
