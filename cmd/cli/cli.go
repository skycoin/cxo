package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
)

const (
	HISTORY = ".cxocli.history"
	ADDRESS = "[::]:8997"
)

var (
	ErrUnknowCommand    = errors.New("unknown command")
	ErrMisisngArgument  = errors.New("missing argument")
	ErrTooManyArguments = errors.New("too many arguments")

	commands = []string{
		"want",
		"got",
		"add_feed",
		"del_feed",
		"feeds",
		"stat",
		"connections",
		"incoming_connections",
		"outgoing_connections",
		"connect",
		"disconnect",
		"listening_address",
		"tree",
		"terminate",
		"quit",
		"exit",
	}
)

func main() {
	var (
		address string
		execute string

		rpc *node.RPCClient
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
		ADDRESS,
		"rpc address")
	flag.StringVar(&execute,
		"e",
		"",
		"execute command and exit")

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

	if rpc, err = node.NewRPCClient(address); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
		return
	}
	defer rpc.Close()

	if execute != "" {
		_, err = executeCommand(execute, rpc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
		return
	}

	line = liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true) // why it is not work

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range commands {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	// load and save history file
	if err = loadHistory(line); err != nil {
		fmt.Fprintln(os.Stderr, "error loading history:", err)
	}
	defer saveHistory(line)

	// rpompt loop

	fmt.Println("enter 'help' to get help")
	for {
		cmd, err = line.Prompt("> ")
		if err != nil && err != liner.ErrPromptAborted {
			fmt.Fprintln(os.Stderr, "fatal error:", err)
			code = 1
			return
		}
		terminate, err = executeCommand(cmd, rpc)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if terminate {
			return
		}
		line.AppendHistory(cmd)
	}

}

func histroyFilePath() (hf string, err error) {
	var usr *user.User
	if usr, err = user.Current(); err != nil {
		return
	}
	hf = filepath.Join(usr.HomeDir, HISTORY)
	return
}

func loadHistory(line *liner.State) (err error) {
	var hf string
	hf, err = histroyFilePath()
	if err != nil {
		return
	}
	var fl *os.File
	if fl, err = os.Open(hf); err != nil {
		if os.IsNotExist(err) {
			err = nil
			return // no history file found
		}
		return
	}
	defer fl.Close()
	_, err = line.ReadHistory(fl)
	return
}

func saveHistory(line *liner.State) {
	var fl *os.File
	hf, err := histroyFilePath()
	if err != nil {
		goto Error
	}
	if fl, err = os.Create(hf); err != nil {
		goto Error
	}
	defer fl.Close()
	if _, err = line.WriteHistory(fl); err != nil {
		goto Error
	}
	return
Error:
	fmt.Fprintln(os.Stderr, "error saving history:", err)
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

func executeCommand(command string, rpc *node.RPCClient) (terminate bool,
	err error) {

	ss := strings.Fields(command)
	if len(ss) == 0 {
		return
	}
	switch strings.ToLower(ss[0]) {
	case "want":
		err = want(rpc, ss)
	case "got":
		err = got(rpc, ss)
	case "add_feed":
		err = add_feed(rpc, ss)
	case "del_feed":
		err = del_feed(rpc, ss)
	case "feeds":
		err = feeds(rpc)
	case "stat":
		err = stat(rpc)
	case "connections":
		err = connections(rpc)
	case "incoming_connections":
		err = incoming_connections(rpc)
	case "outgoing_connections":
		err = outgoing_connections(rpc)
	case "connect":
		err = connect(rpc, ss)
	case "disconnect":
		err = disconnect(rpc, ss)
	case "listening_address":
		err = listening_address(rpc)
	case "tree":
		err = tree(rpc, ss)
	case "terminate":
		err = term(rpc)
	// help and exit
	case "help":
		showHelp()
	case "quit", "exit":
		terminate = true
		fmt.Println("cya")
	default:
		err = ErrUnknowCommand
	}
	return
}

func showHelp() {
	fmt.Println(`

  want <public key>
    list objects of feed the server doesn't have (yet), but knows about
  got <public key>
    list objects of feed
  add_feed <public key>
    start shareing feed
  del_feed <public key>
    stop sharing feed
  feeds
    list feeds
  stat
    database statistic
  connections
    list connections
  incoming_connections
    list incoming connections
  outgoing_connections
    list outgoing connections
  connect <address>
    connect to
  disconnect <address>
    disconnect from
  listening_address
    print listening address
  tree <public key>
    print objects tree of feed
  terminate
    terminate server if allowed
  help
    show this help message
  quit or exit
    leave the cli

`)
}

// ========================================================================== //
//                            cxo related commands                            //
// ========================================================================== //

// helper
func publicKeyArg(ss []string) (pub cipher.PubKey, err error) {
	var pubs string
	if pubs, err = args(ss); err != nil {
		return
	}
	pub, err = cipher.PubKeyFromHex(pubs)
	return
}

func want(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	var list []cipher.SHA256
	if list, err = rpc.Want(pk); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Println("  don't want anything")
		return
	}
	for _, w := range list {
		fmt.Println("  +", w.Hex())
	}
	return
}

func got(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	var list []cipher.SHA256
	if list, err = rpc.Got(pk); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Println("  hasn't got anything")
		return
	}
	for _, w := range list {
		fmt.Println("  +", w.Hex())
	}
	return
}

func add_feed(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	var ok bool
	if ok, err = rpc.AddFeed(pk); err != nil {
		return
	}
	if ok {
		fmt.Println("  success")
		return
	}
	fmt.Println("  already")
	return
}

func del_feed(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	var ok bool
	if ok, err = rpc.DelFeed(pk); err != nil {
		return
	}
	if ok {
		fmt.Println("  success")
		return
	}
	fmt.Println("  hasn't got the feed")
	return
}

func feeds(rpc *node.RPCClient) (err error) {
	var list []cipher.PubKey
	if list, err = rpc.Feeds(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Println("  no feeds")
		return
	}
	for _, f := range list {
		fmt.Println("  +", f.Hex())
	}
	return
}

func stat(rpc *node.RPCClient) (err error) {
	var stat data.Stat
	if stat, err = rpc.Stat(); err != nil {
		return
	}
	fmt.Println("  ----")
	fmt.Println("  Objects:", stat.Objects)
	fmt.Println("  Space:  ", stat.Space.String())
	fmt.Println("  ----")
	for pk, fs := range stat.Feeds {
		fmt.Println("  -", pk.Hex())
		fmt.Println("    Root Objects: ", fs.Roots)
		fmt.Println("    Space:        ", fs.Space.String())
	}
	if len(stat.Feeds) > 0 {
		fmt.Println("  ----")
	}
	return
}

func connections(rpc *node.RPCClient) (err error) {
	var list []string
	if list, err = rpc.Connections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Println("  no connections")
		return
	}
	for _, c := range list {
		fmt.Println("  +", c)
	}
	return
}

func incoming_connections(rpc *node.RPCClient) (err error) {
	var list []string
	if list, err = rpc.IncomingConnections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Println("  no incoming connections")
		return
	}
	for _, c := range list {
		fmt.Println("  +", c)
	}
	return
}

func outgoing_connections(rpc *node.RPCClient) (err error) {
	var list []string
	if list, err = rpc.OutgoingConnections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Println("  no outgoing connections")
		return
	}
	for _, c := range list {
		fmt.Println("  +", c)
	}
	return
}

func connect(rpc *node.RPCClient, ss []string) (err error) {
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

func disconnect(rpc *node.RPCClient, ss []string) (err error) {
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

func listening_address(rpc *node.RPCClient) (err error) {
	var address string
	if address, err = rpc.ListeningAddress(); err != nil {
		return
	}
	fmt.Println("  ", address)
	return
}

func tree(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	var tree []byte
	if tree, err = rpc.Tree(pk); err != nil {
		return
	}
	if len(tree) == 0 {
		fmt.Println("  empty tree")
		return
	}
	_, err = os.Stdout.Write(tree)
	return
}

func term(rpc *node.RPCClient) (err error) {
	if err = rpc.Terminate(); err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil {
		return
	}
	fmt.Println("  terminated")
	return
}
