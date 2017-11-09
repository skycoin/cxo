package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/peterh/liner"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	cxo "github.com/skycoin/cxo/skyobject"
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
	errInvalidQuery     = errors.New("invalid query")

	commands = []string{
		"add feed ",
		"del feed ",

		"subscribe ",
		"unsubscribe ",

		"connect ",
		"disconnect ",

		"connections ",
		"incoming connections ",
		"outgoing connections ",

		"connection ",
		"feed ",

		"feeds ",

		"roots ",
		"tree ",

		"info ",
		"listening address ",

		"stat ",

		"terminate ",

		"quit ",
		"exit ",
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

	defer func() {
		// so the os.Exit "recovers" silently
		// that is not acceptable for developers,
		// we have to handle panicing
		if err := recover(); err != nil {
			panic(err)
		}
		os.Exit(code)
	}()

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

	// prompt loop

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
		err = subscribeTo(rpc, ss)
	case "unsubscribe":
		err = unsubscribeFrom(rpc, ss)

	case "connect":
		err = connect(rpc, ss)
	case "disconnect":
		err = disconnect(rpc, ss)

	case "connections":
		err = connections(rpc)

	case "connection":
		err = connection(rpc, ss)
	case "feed":
		err = feed(rpc, ss)

	case "feeds":
		err = feeds(rpc)

	case "roots":
		err = roots(rpc, ss)
	case "tree":
		err = tree(rpc, ss)

	case "info":
		err = info(rpc)

	case "stat":
		err = stat(rpc, ss)

	case "terminate":
		err = term(rpc)

	case "help":
		showHelp()

	case "quit", "exit":
		terminate = true
		fmt.Fprintln(out, "cya")

	default:

		if len(ss) < 2 {
			err = errUnknowCommand
			break
		}

		switch ss[0] + " " + ss[1] {

		case "add feed":
			err = addFeed(rpc, ss)
		case "del feed":
			err = delFeed(rpc, ss)

		case "incoming connections":
			err = incomingConnections(rpc)
		case "outgoing connections":
			err = outgoingConnections(rpc)

		case "listening address":
			err = listeningAddress(rpc)

		default:
			err = errUnknowCommand
		}

	}
	return
}

func showHelp() {
	fmt.Fprintln(out, `

  add feed <public key>
    start shareing feed
  del feed <public key>
    stop sharing feed

  subscribe <address> <pub key>
    subscribe to feed of a connected peer
  unsubscribe <address> <public key>
    unsubscribe from feed of a connected peer

  connect <address>
    connect to node with given address
  disconnect <address>
    disconnect from given address

  connections
    list all connections, in the list "-->" means that
    this connection is incoming and "<--" means that this
    connection is outgoing; (✓) means that connection is
    established, and (⌛) means that this connection is
    establishing
  incoming connections
    list all incoming connections
  outgoing connections
    list all outgoing connections

  connection <address>
    list feeds of given connection
  feed <public key>
    list connections of given feed

  feeds
    list all feeds

  roots <public key>
    print brief information about all root objects of given feed
  tree <pub key> [seq]
    print root by public key and seq number, if the seq omitted then
    last full root printed

  info
    get brief information about node
  listening address
    print listening address

  stat [public key] [seq query]
    show detailed statistic; optional public key shows statistic
    of a feed (instead of all feeds), and optional seq query
    shows statistic of range of root objects; the stat can be used
    this ways:

    stat
      statistic of all, by default it shows statistic of every feed
      and last root of every feed
    stat full
      statistic of all, output can be big
    stat 022241cd1857ec73b7741ccb5d5494743e1ef7075ec71d6eca59979fd6be58758c
      statistic of given feed

    the "seq query" can be used if public key given, there are variants:
      1    - statistic of given root (with seq = 1)
      1..3 - statistic of root objects 1, 2 and 3
      ..3  - statistic of all root objects before 3 (inclusive)
      3..  - statistic of all root objects after 3 (inclusive)
      -3   - statistic of latest 3 root objects
      +3   - statistic of first 3 root objects

    for example

    stat 022241cd1857ec73b7741ccb5d5494743e1ef7075ec71d6eca59979fd6be58758c 1..3

  terminate
    terminate server if allowed

  help
    show this help message

  quit, exit
    leave the cli

`)
}

// ========================================================================== //
//                            cxo related commands                            //
// ========================================================================== //

func pubKeyFromHex(pks string) (pk cipher.PubKey, err error) {
	var b []byte
	if b, err = hex.DecodeString(pks); err != nil {
		return
	}
	if len(b) != len(cipher.PubKey{}) {
		err = errors.New("invalid PubKey length")
	}
	pk = cipher.NewPubKey(b)
	return
}

// helper
func publicKeyArg(ss []string) (pub cipher.PubKey, err error) {
	var pubs string
	if pubs, err = args(ss); err != nil {
		return
	}
	pub, err = pubKeyFromHex(pubs)
	return
}

func addFeed(rpc *node.RPCClient, ss []string) (err error) {
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
		pk, err = pubKeyFromHex(ss[2])
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

func delFeed(rpc *node.RPCClient, ss []string) (err error) {
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

func connection(rpc *node.RPCClient, ss []string) (err error) {
	var address string
	if address, err = args(ss); err != nil {
		return
	}
	var feeds []cipher.PubKey
	if feeds, err = rpc.Connection(address); err != nil {
		return
	}
	if len(feeds) == 0 {
		fmt.Fprintln(out, "  no feeds")
		return
	}
	for _, f := range feeds {
		fmt.Fprintln(out, "  +", f.Hex())
	}
	return
}

func feed(rpc *node.RPCClient, ss []string) (err error) {
	var pk cipher.PubKey
	if pk, err = publicKeyArg(ss); err != nil {
		return
	}
	var conns []string
	if conns, err = rpc.Feed(pk); err != nil {
		return
	}
	if len(conns) == 0 {
		fmt.Fprintln(out, "  no connections")
		return
	}
	for _, c := range conns {
		fmt.Fprintln(out, "  +", c)
	}
	return
}

func stat(rpc *node.RPCClient, args []string) (err error) {

	var stat node.Stat
	if stat, err = rpc.Stat(); err != nil {
		return
	}

	fmt.Fprintln(out, "  ----")
	fmt.Fprintln(out, "  Connections:", stat.Node.Connections)
	fmt.Fprintln(out, "  Feeds:      ", stat.Node.Feeds)
	fmt.Fprintln(out, "  Filling avg:", stat.Node.FillAvg)
	fmt.Fprintln(out, "  Drop avg:   ", stat.Node.DropAvg)
	fmt.Fprintln(out)
	fmt.Fprintln(out, "  Save avg:    ", stat.CXO.Save)
	fmt.Fprintln(out, "  Clean up avg:", stat.CXO.CleanUp)
	fmt.Fprintln(out, "  Stat time:   ", stat.CXO.Stat)
	fmt.Fprintln(out, "  ----")

	sap, svp := stat.CXO.Percents()

	fmt.Fprintf(out, "  Total objects amount: %d (%.2f%% shared)\n",
		stat.CXO.Objects.Amount, sap*100.0)
	fmt.Fprintf(out, "  Total objects volume: %s (%.2f%% shared)\n",
		stat.CXO.Objects.Volume, svp*100.0)

	if len(stat.CXO.Feeds) == 0 {
		fmt.Fprintln(out, "  no feeds")
		fmt.Fprintln(out, "  ----")
		return // don't give a shit about arguments and anything else
	}

	// 1. no arguments    -> print every feed with last root
	// 2. 'full' argument -> print all
	// 3. pk argument     -> print given feed
	// 4. pk + query      -> print by the query

	switch len(args) {
	case 0, 1:
		// no arguments
		printDefaultFeedsStat(&stat.CXO)
	case 2:
		// one argument (full or pk)
		if args[1] == "full" {
			printFullFeedsStat(&stat.CXO)
		} else {
			var pk cipher.PubKey
			if pk, err = pubKeyFromHex(args[1]); err != nil {
				return
			}
			if fs, ok := stat.CXO.Feeds[pk]; ok {
				printFullFeedStat(pk, &fs)
				return
			}
			fmt.Fprintln(out, "  no such feed")
		}
	case 3:
		// pk + query
		var pk cipher.PubKey
		if pk, err = pubKeyFromHex(args[1]); err != nil {
			return
		}
		if fs, ok := stat.CXO.Feeds[pk]; ok {
			err = printFeedStatQuery(pk, &fs, args[2])
			return
		}
		fmt.Fprintln(out, "  no such feed")
	default:
		return errTooManyArguments
	}

	fmt.Fprintln(out, "  ----")
	return
}

func printFirstRoots(query string, fs *cxo.FeedStat) (err error) {
	var num uint64
	if num, err = strconv.ParseUint(query, 10, 64); err != nil {
		return
	}
	crs := sortRootStats(fs)
	for i := uint64(0); i < num && i < uint64(len(fs.Roots)); i++ {
		printRootStat(crs[i].seq, crs[i].stat)
	}
	return
}

func printLastRoots(query string, fs *cxo.FeedStat) (err error) {
	var num uint64
	if num, err = strconv.ParseUint(query, 10, 64); err != nil {
		return
	}
	crs := sortRootStats(fs)
	last := uint64(len(fs.Roots)) - 1
	for i := last; i >= 0 && i > last-num; i-- {
		printRootStat(crs[i].seq, crs[i].stat)
	}
	return
}

func printRootsBefore(query string, fs *cxo.FeedStat) (err error) {
	var before uint64
	if before, err = strconv.ParseUint(query, 10, 64); err != nil {
		return
	}
	var printed bool
	for _, cr := range sortRootStats(fs) {
		if cr.seq > before {
			break
		}
		printed = true
		printRootStat(cr.seq, cr.stat)
	}
	if false == printed {
		fmt.Fprintln(out, "      no such root objects")
	}
	return
}

func printRootsAfter(query string, fs *cxo.FeedStat) (err error) {
	var after uint64
	if after, err = strconv.ParseUint(query, 10, 64); err != nil {
		return
	}
	var printed bool
	for _, cr := range sortRootStats(fs) {
		if cr.seq < after {
			continue
		}
		printed = true
		printRootStat(cr.seq, cr.stat)
	}
	if false == printed {
		fmt.Fprintln(out, "      no such root objects")
	}
	return
}

func printRootsRange(query string, fs *cxo.FeedStat) (err error) {
	switch {
	case strings.HasPrefix(query, ".."):
		// before (inclusive)
		return printRootsBefore(strings.TrimPrefix(query, ".."), fs)
	case strings.HasSuffix(query, ".."):
		// after (inclusive)
		return printRootsAfter(strings.TrimSuffix(query, ".."), fs)
	default:
		// n..m
		ss := strings.Split(query, "..")
		if len(ss) != 2 {
			return errInvalidQuery
		}
		var after, before uint64
		if after, err = strconv.ParseUint(ss[0], 10, 64); err != nil {
			return
		}
		if before, err = strconv.ParseUint(ss[1], 10, 64); err != nil {
			return
		}
		if before < after {
			return errInvalidQuery
		}
		var printed bool
		for _, cr := range sortRootStats(fs) {
			if cr.seq < after {
				continue
			}
			if cr.seq > before {
				break
			}
			printed = true
			printRootStat(cr.seq, cr.stat)
		}
		if false == printed {
			fmt.Fprintln(out, "      no such root objects")
		}
	}
	return
}

func printFeedStatQuery(pk cipher.PubKey, fs *cxo.FeedStat,
	query string) (err error) {

	// 1    - given root (with seq = 1)
	// 1..3 - root objects 1, 2 and 3
	// ..3  - all root objects before 3 (inclusive)
	// 3..  - all root objects after 3 (inclusive)
	// -3   - latest 3 root objects
	// +3   - first 3 root objects

	printFeedStatHead(pk, fs)
	if len(fs.Roots) == 0 {
		fmt.Fprintln(out, "      no root objects")
		return
	}

	switch {
	case strings.Contains(query, ".."):
		// ..3
		// 3..
		// 1..3
		err = printRootsRange(query, fs)
	case strings.HasPrefix(query, "+"):
		// +3
		err = printFirstRoots(strings.TrimPrefix(query, "+"), fs)
	case strings.HasPrefix(query, "-"):
		// -3
		err = printLastRoots(strings.TrimPrefix(query, "-"), fs)
	default:
		var seq uint64
		if seq, err = strconv.ParseUint(query, 10, 64); err != nil {
			return
		}
		if rs, ok := fs.Roots[seq]; ok {
			printRootStat(seq, &rs)
			return
		}
		fmt.Fprintln(out, "      no such root object")
	}
	return
}

func printFeedStatHead(pk cipher.PubKey, fs *cxo.FeedStat) {
	fmt.Fprintln(out, "  -", pk.Hex())

	sap, svp := fs.Percents()

	fmt.Fprintf(out, "    total objects amount: %d (%.2f%% shared)\n",
		fs.Objects.Amount, sap*100.0)
	fmt.Fprintf(out, "    total objects volume: %s (%.2f%% shared)\n",
		fs.Objects.Volume, svp*100.0)

}

func printRootStat(seq uint64, rs *cxo.RootStat) {
	fmt.Fprintln(out, "    -", seq)

	sap, svp := rs.Percents()

	fmt.Fprintf(out, "      total objects amount: %d (%.2f%% shared)\n",
		rs.Objects.Amount, sap*100.0)
	fmt.Fprintf(out, "      total objects volume: %s (%.2f%% shared)\n",
		rs.Objects.Volume, svp*100.0)
}

func printDefaultFeedsStat(stat *cxo.Stat) {
	cfs := sortFeedStats(stat.Feeds)
	for _, fs := range cfs {
		printFeedStatHead(fs.pk, fs.stat)
		if len(fs.stat.Roots) == 0 {
			fmt.Fprintln(out, "      no root objects")
			continue
		}
		seq, rs := getLastRootStat(fs.stat)
		printRootStat(seq, rs)
	}
}

func printFullFeedStat(pk cipher.PubKey, fs *cxo.FeedStat) {
	printFeedStatHead(pk, fs)
	if len(fs.Roots) == 0 {
		fmt.Fprintln(out, "      no root objects")
		return
	}
	for _, crs := range sortRootStats(fs) {
		printRootStat(crs.seq, crs.stat)
	}
}

func printFullFeedsStat(stat *cxo.Stat) {
	cfs := sortFeedStats(stat.Feeds)
	for _, fs := range cfs {
		printFullFeedStat(fs.pk, fs.stat)
	}
}

func connDirString(in bool) string {
	if in {
		return "-->"
	}
	return "<--"
}

func connPendString(pend bool) string {
	if pend {
		return "(⌛)"
	}
	return "(✓)"
}

func nodeConnectionString(nc node.ConnectionInfo) string {
	return fmt.Sprintf("%s %s %s", connDirString(nc.IsIncoming),
		nc.Address, connPendString(nc.IsPending))
}

func connections(rpc *node.RPCClient) (err error) {
	var list []node.ConnectionInfo
	if list, err = rpc.Connections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no connections")
		return
	}
	for _, c := range list {
		fmt.Fprintln(out, " ", nodeConnectionString(c))
	}
	return
}

func incomingConnections(rpc *node.RPCClient) (err error) {
	var list []node.ConnectionInfo
	if list, err = rpc.IncomingConnections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no incoming connections")
		return
	}
	for _, c := range list {
		fmt.Fprintf(out, "  %s %s\n", c.Address, connPendString(c.IsPending))
	}
	return
}

func outgoingConnections(rpc *node.RPCClient) (err error) {
	var list []node.ConnectionInfo
	if list, err = rpc.OutgoingConnections(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no outgoing connections")
		return
	}
	for _, c := range list {
		fmt.Fprintf(out, "  %s %s\n", c.Address, connPendString(c.IsPending))
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

func info(rpc *node.RPCClient) (err error) {
	var info *node.Info
	if info, err = rpc.Info(); err != nil {
		return
	}
	fmt.Fprintln(out, "  ----")
	fmt.Fprintf(out, "  listening: %v\n", info.IsListening)
	fmt.Fprintln(out, "  address:  ", info.ListeningAddress)
	fmt.Fprintln(out, "  discovery:", info.Discovery)
	fmt.Fprintf(out, "  public:    %v\n", info.IsPublicServer)
	fmt.Fprintln(out, "  ----")
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
		fmt.Fprintln(out, "      time:       ", ri.Time.Format(time.ANSIC))
		fmt.Fprintln(out, "      seq:        ", ri.Seq)
		var prev string
		if ri.Prev == (cipher.SHA256{}) {
			prev = "(blank)"
		} else {
			prev = ri.Prev.Hex()[:7]
		}
		fmt.Fprintln(out, "      prev:       ", prev)
		fmt.Fprintln(out, "      created at: ",
			ri.CreateTime.Format(time.ANSIC))
		fmt.Fprintln(out, "      last access:",
			ri.AccessTime.Format(time.ANSIC))
		fmt.Fprintln(out, "      refs count: ", ri.RefsCount)
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
	if pk, err = pubKeyFromHex(ss[1]); err != nil {
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

// --- sorting (feeds)

func sortFeedStats(fss map[cipher.PubKey]cxo.FeedStat) (cfs cxoFeedStats) {
	cfs = make(cxoFeedStats, 0, len(fss))
	for pk, fs := range fss {
		cfs = append(cfs, cxoFeedStat{
			pk:   pk,
			stat: &fs,
		})
	}
	sort.Sort(cfs)
	return
}

type cxoFeedStat struct {
	pk   cipher.PubKey
	stat *cxo.FeedStat
}

type cxoFeedStats []cxoFeedStat

func (c cxoFeedStats) Len() int { return len(c) }
func (c cxoFeedStats) Less(i, j int) bool {
	return bytes.Compare(c[i].pk[:], c[j].pk[:]) < 0
}
func (c cxoFeedStats) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

// --- sorting (roots)

func getLastRootStat(fs *cxo.FeedStat) (last uint64, rs *cxo.RootStat) {
	for seq, crs := range fs.Roots {
		if seq > last || (seq == 0 && last == 0) {
			rs = &crs
			last = seq
		}
	}
	return // or nil
}

func sortRootStats(fs *cxo.FeedStat) (rss cxoRootStats) {
	rss = make(cxoRootStats, 0, len(fs.Roots))
	for seq, rs := range fs.Roots {
		rss = append(rss, cxoRootStat{
			seq:  seq,
			stat: &rs,
		})
	}
	sort.Sort(rss)
	return
}

type cxoRootStat struct {
	seq  uint64
	stat *cxo.RootStat
}

type cxoRootStats []cxoRootStat

func (c cxoRootStats) Len() int           { return len(c) }
func (c cxoRootStats) Less(i, j int) bool { return c[i].seq < c[j].seq }
func (c cxoRootStats) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }
