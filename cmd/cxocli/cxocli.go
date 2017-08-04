package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/peterh/liner"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
)

// defaults
const (
	HISTORY = ".cxocli.history" // history file name
	ADDRESS = "[::]:8997"       // default RPC address to connect to
)

var (
	out io.Writer = os.Stdout // it's nessesary for tests

	errUnknowCommand    = errors.New("unknown command")
	errMisisngArgument  = errors.New("missing argument")
	errTooManyArguments = errors.New("too many arguments")

	commands = []string{
		"subscribe",
		"subscribe_to",
		"unsubscribe",
		"unsubscribe_from",
		"feeds",
		"stat",
		"connections",
		"incoming_connections",
		"outgoing_connections",
		"connect",
		"disconnect",
		"listening_address",
		"roots",
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
		fmt.Fprintf(out, "Usage %s <flags>\n", os.Args[0])
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

	fmt.Fprintln(out, "enter 'help' to get help")
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
		return "", errMisisngArgument
	case 2:
		return ss[1], nil
	}
	return "", errTooManyArguments
}

func executeCommand(command string, rpc *node.RPCClient) (terminate bool,
	err error) {

	ss := strings.Fields(command)
	if len(ss) == 0 {
		return
	}
	switch strings.ToLower(ss[0]) {
	case "subscribe":
		err = subscribe(rpc, ss)
	case "subscribe_to":
		err = subscribeTo(rpc, ss)
	case "unsubscribe":
		err = unsubscribe(rpc, ss)
	case "unsubscribe_from":
		err = unsubscribeFrom(rpc, ss)
	case "feeds":
		err = feeds(rpc)
	case "stat":
		err = stat(rpc)
	case "connections":
		err = connections(rpc)
	case "incoming_connections":
		err = incomingConnections(rpc)
	case "outgoing_connections":
		err = outgoingConnections(rpc)
	case "connect":
		err = connect(rpc, ss)
	case "disconnect":
		err = disconnect(rpc, ss)
	case "listening_address":
		err = listeningAddress(rpc)
	case "roots":
		err = roots(rpc, ss)
	case "tree":
		err = tree(rpc, ss)
	case "terminate":
		err = term(rpc)
	// help and exit
	case "help":
		showHelp()
	case "quit", "exit":
		terminate = true
		fmt.Fprintln(out, "cya")
	default:
		err = errUnknowCommand
	}
	return
}

func showHelp() {
	fmt.Fprintln(out, `

  subscribe <public key>
    start shareing feed
  subscribe_to <address> <pub key>
    subscribe to feed of a connected peer
  unsubscribe <public key>
    stop sharing feed
  unsubscribe_from <address> <public key>
    unsubscribe_from feed of a connected peer
  feeds
    list feeds
  stat
    statistic
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
  roots <public key>
    print brief information about all root objects of given feed
  tree <pub key> [seq]
    print root by public key and seq number, if the seq omited then
    last full root printed
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

func subscribe(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	if err = rpc.Subscribe("", pk); err != nil {
		return
	}
	fmt.Fprintln(out, "  subscribed")
	return
}

func addressFeedArgs(ss []string) (address string, pk cipher.PubKey,
	err error) {

	switch len(ss) {
	case 0, 1, 2:
		err = errMisisngArgument
	case 3:
		address = ss[1]
		pk, err = cipher.PubKeyFromHex(ss[2])
	default:
		err = errTooManyArguments
	}
	return
}

func subscribeTo(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	var address string
	if address, pk, err = addressFeedArgs(ss); err != nil {
		return
	}
	if err = rpc.Subscribe(address, pk); err != nil {
		return
	}
	fmt.Fprintln(out, "  subscribed") // optimistic
	return
}

func unsubscribe(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	if err = rpc.Unsubscribe("", pk); err != nil {
		return
	}
	fmt.Fprintln(out, "  unsubscribed") // optimistic
	return
}

func unsubscribeFrom(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	var address string
	if address, pk, err = addressFeedArgs(ss); err != nil {
		return
	}
	if err = rpc.Unsubscribe(address, pk); err != nil {
		return
	}
	fmt.Fprintln(out, "  unsubscribed") // optimistic
	return
}

func feeds(rpc *node.RPCClient) (err error) {
	var list []cipher.PubKey
	if list, err = rpc.Feeds(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no feeds")
		return
	}
	for _, f := range list {
		fmt.Fprintln(out, "  +", f.Hex())
	}
	return
}

func stat(rpc *node.RPCClient) (err error) {
	var stat node.Stat
	if stat, err = rpc.Stat(); err != nil {
		return
	}
	fmt.Fprintln(out, "  ----")
	fmt.Fprintln(out, "  Objects:", stat.Data.Objects)
	fmt.Fprintln(out, "  Space:  ", stat.Data.Space.String())
	fmt.Fprintln(out, "  ----")
	for pk, fs := range stat.Data.Feeds {
		fmt.Fprintln(out, "  -", pk.Hex())
		fmt.Fprintln(out, "    Root Objects: ", fs.Roots)
		fmt.Fprintln(out, "    Space:        ", fs.Space.String())
	}
	fmt.Fprintln(out, "  ----")
	fmt.Fprintln(out, "  Registries:    ", stat.CXO.Registries)
	fmt.Fprintln(out, "  Save    (avg): ", stat.CXO.Save)
	fmt.Fprintln(out, "  Cleanup (avg): ", stat.CXO.CleanUp)
	fmt.Fprintln(out, "  ----")
	return
}

func connections(rpc *node.RPCClient) (err error) {
	var list []string
	if list, err = rpc.Connections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no connections")
		return
	}
	for _, c := range list {
		fmt.Fprintln(out, "  +", c)
	}
	return
}

func incomingConnections(rpc *node.RPCClient) (err error) {
	var list []string
	if list, err = rpc.IncomingConnections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no incoming connections")
		return
	}
	for _, c := range list {
		fmt.Fprintln(out, "  +", c)
	}
	return
}

func outgoingConnections(rpc *node.RPCClient) (err error) {
	var list []string
	if list, err = rpc.OutgoingConnections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no outgoing connections")
		return
	}
	for _, c := range list {
		fmt.Fprintln(out, "  +", c)
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
	fmt.Fprintln(out, "  connected")
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
	fmt.Fprintln(out, "  disconnected")
	return
}

func listeningAddress(rpc *node.RPCClient) (err error) {
	var address string
	if address, err = rpc.ListeningAddress(); err != nil {
		return
	}
	fmt.Fprintln(out, "  ", address)
	return
}

func roots(rpc *node.RPCClient, ss []string) (err error) {
	var pub cipher.PubKey
	if pub, err = publicKeyArg(ss); err != nil {
		return
	}
	var ris []node.RootInfo
	if ris, err = rpc.Roots(pub); err != nil {
		return
	}
	if len(ris) == 0 {
		fmt.Fprintln(out, "  empty feed")
		return
	}
	for _, ri := range ris {
		fmt.Fprintln(out, "  -", ri.Hash.Hex())
		fmt.Fprintln(out, "      time:", ri.Time)
		fmt.Fprintln(out, "      seq:", ri.Seq)
		fmt.Fprintln(out, "      fill:", ri.IsFull)
	}
	return
}

func tree(rpc *node.RPCClient, ss []string) (err error) {

	var pk cipher.PubKey
	var seq uint64
	var lsatFull bool

	switch len(ss) {
	case 0, 1:
		return errors.New("to few arguments: want <pub key> [seq]")
	case 2:
		lsatFull = true
	case 3:
	default:
		return errors.New("to many arguments: want <pub key> [seq]")
	}
	if pk, err = cipher.PubKeyFromHex(ss[1]); err != nil {
		return
	}
	if lsatFull == false {
		if seq, err = strconv.ParseUint(ss[2], 10, 64); err != nil {
			return
		}
	}
	var tree string
	if tree, err = rpc.Tree(pk, seq, lsatFull); err != nil {
		return
	}
	fmt.Fprintln(out, tree)
	return
}

func term(rpc *node.RPCClient) (err error) {
	if err = rpc.Terminate(); err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err != nil {
		return
	}
	fmt.Fprintln(out, "  terminated")
	return
}
