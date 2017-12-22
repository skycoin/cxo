package main

import (
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/peterh/liner"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/node"
	"github.com/skycoin/cxo/skyobject"
	"github.com/skycoin/cxo/skyobject/registry"
)

// defaults
const (
	HISTORY = ".cxocli.history" // history file name
	ADDRESS = "[::]:8871"       // default RPC address to connect to
)

var (
	out io.Writer = os.Stdout // it's necessary for tests

	errUnknowCommand    = errors.New("unknown command")
	errMisisngArgument  = errors.New("missing argument")
	errTooManyArguments = errors.New("too many arguments")
	errInvalidQuery     = errors.New("invalid query")

	commands = []string{

		// feeds

		"share feed ",
		"don't share feed ",
		"list feeds ",
		"is shareing ",

		// tcp

		"tcp connect ",
		"tcp disconnect ",

		"tcp subsribe ",
		"tcp unsubscribe ",

		"tcp address ",

		// udp

		"udp connect ",
		"udp disconnect ",

		"udp subscribe ",
		"udp unsubscribe ",

		"udp address ",

		// all connections

		"connections ",
		"connections of feed ",

		// root objects

		"root info ",
		"root tree ",
		"last root ",

		// stat

		"stat ",

		// help

		"help",

		// leave the cli

		"quit ",
		"exit ",
	}
)

func main() {

	var (
		address string
		execute string

		rpc = new(client)
		err error

		line *liner.State
		cmd  string
		quit bool

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

	if rpc.r, err = node.NewRPCClient(address); err != nil {
		fmt.Fprintln(os.Stderr, err)
		code = 1
		return
	}
	defer rpc.r.Close()

	if execute != "" {
		_, err = rpc.executeCommand(execute)
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

	fmt.Fprintln(out, "enter 'help' to get help, use 'tab' to complite command")
	for {
		cmd, err = line.Prompt("> ")
		if err != nil && err != liner.ErrPromptAborted {
			fmt.Fprintln(os.Stderr, "fatal error:", err)
			code = 1
			return
		}
		quit, err = rpc.executeCommand(cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if quit {
			return
		}
		line.AppendHistory(cmd)
	}

}

func histroyFilePath() (hf string, err error) {
	hf = filepath.Join(skyobject.DataDir(), HISTORY)
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
		if os.IsNotExist(err) == true {
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

type client struct {
	r *node.RPCClient
	m map[string]func(in []string) (err error)

	// TODO (kostyarin): autocomplite feeds, nonces, seq numbers,
	//                   connections
}

func (c *client) mapping() map[string]func(in []string) (err error) {

	if c.m != nil {
		return c.m
	}

	c.m = map[string]func(in []string) (err error){
		"share feed":       c.share,
		"don't share feed": c.dontShare,
		"list feeds":       c.listFeeds,
		"is shareing":      c.isShareing,

		"tcp connect":     c.tcpConnect,
		"tcp disconnect":  c.tcpDisconnet,
		"tcp subsribe":    c.tcpSubscribe,
		"tcp unsubscribe": c.tcpUnsubscribe,
		"tcp address":     c.tcpAddress,

		"udp connect":     c.udpConnect,
		"udp disconnect":  c.udpDisconnet,
		"udp subsribe":    c.udpSubscribe,
		"udp unsubscribe": c.udpUnsubscribe,
		"udp address":     c.udpAddress,

		"connections":         c.connections,
		"connections of feed": c.connectionsOfFeed,

		"root info": c.rootInfo,
		"root tree": c.rootTree,
		"last root": c.lastRoot,

		"stat": c.stat,

		"help": c.help,

		"quit": c.quit,
		"exit": c.quit,
	}

	return c.m

}

func prefixLength(in, pfx string) (pl int) {
	var ml = len(pfx)

	if len(in) < ml {
		ml = len(in)
	}

	for pl = 0; pl < ml && in[pl] == pfx[pl]; pl++ {
	}

	return
}

func trimPrefix(in, pfx string) (out []string) {
	return strings.Fields(strings.TrimSpace(strings.TrimPrefix(in, pfx)))
}

func (c *client) executeCommand(in string) (quit bool, err error) {

	var prefix string // longest prefix

	for cmd := range c.mapping() {
		if pl := prefixLength(in, cmd); pl > len(prefix) {
			prefix = in[:pl]
		}
	}

	prefix = strings.TrimSpace(prefix)

	var fn, ok = c.mapping()[prefix]

	if ok == false {
		return false, fmt.Errorf("unknown command %q", in)
	}

	if prefix == "quit" || prefix == "exit" {
		quit = true
	}

	err = fn(trimPrefix(in, prefix))
	return
}

func (c *client) argsOne(
	in []string,
	name string,
) (
	one string,
	err error,
) {

	if len(in) == 0 {
		return "", errors.New("missing argument: expected " + name)
	}

	if len(in) > 1 {
		return "", errors.New("too many arguments, expected " + name + " only")

	}

	return in[0], nil

}

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

func (c *client) argsFeed(in []string) (pk cipher.PubKey, err error) {

	var one string
	if one, err = c.argsOne(in, "public key"); err != nil {
		return
	}

	return pubKeyFromHex(one)
}

func (c *client) argsAddress(in []string) (a string, err error) {
	return c.argsOne(in, "network address")
}

func (c *client) argsConnFeed(in []string) (af node.ConnFeed, err error) {

	const expected = "expected address and public key"

	switch len(in) {
	case 0:
		err = errors.New("missing arguments: " + expected)
	case 1:
		err = errors.New("missing public key: " + expected)
	case 2:
		af.Address = in[0]
		af.Feed, err = pubKeyFromHex(in[1])
	default:
		err = errors.New("too many arguments: " + expected)
	}

	return

}

func (c *client) argsRoot(in []string) (rs node.RootSelector, err error) {

	const expected = "expected public key, nonce and seq number"

	switch len(in) {
	case 0, 1, 2:
		err = errors.New("missing arguments: " + expected)
	case 3:
		if rs.Feed, err = pubKeyFromHex(in[0]); err != nil {
			return
		}
		if rs.Nonce, err = strconv.ParseUint(in[1], 10, 64); err != nil {
			return
		}
		rs.Seq, err = strconv.ParseUint(in[2], 10, 64)
	default:
		err = errors.New("too many arguments: " + expected)
	}

	return

}

func (c *client) argsNo(in []string) (err error) {
	if len(in) != 0 {
		err = errors.New("unexpected arguments, expected nothing")
	}
	return
}

//
// feeds
//

func (c *client) share(in []string) (err error) {
	var pk cipher.PubKey
	if pk, err = c.argsFeed(in); err != nil {
		return
	}
	return c.r.Node().Share(pk)
}

func (c *client) dontShare(in []string) (err error) {
	var pk cipher.PubKey
	if pk, err = c.argsFeed(in); err != nil {
		return
	}
	return c.r.Node().DontShare(pk)
}

func (c *client) listFeeds(in []string) (err error) {
	if err = c.argsNo(in); err != nil {
		return
	}
	var list []cipher.PubKey
	if list, err = c.r.Node().Feeds(); err != nil {
		return
	}
	if len(list) == 0 {
		fmt.Fprintln(out, "  no feeds are shared")
		return
	}
	for _, pk := range list {
		fmt.Fprintln(out, "  -", pk.Hex())
	}
	return
}

func (c *client) isShareing(in []string) (err error) {
	var pk cipher.PubKey
	if pk, err = c.argsFeed(in); err != nil {
		return
	}
	var yep bool
	if yep, err = c.r.Node().IsSharing(pk); err != nil {
		return
	}
	if yep == true {
		fmt.Fprintln(out, "  yes, it is")
		return
	}
	fmt.Fprintln(out, "  no, it is not")
	return
}

//
// tcp
//

func (c *client) tcpConnect(in []string) (err error) {
	var address string
	if address, err = c.argsAddress(in); err != nil {
		return
	}
	return c.r.TCP().Connect(address)
}

func (c *client) tcpDisconnet(in []string) (err error) {
	var address string
	if address, err = c.argsAddress(in); err != nil {
		return
	}
	return c.r.TCP().Disconnect(address)
}

func (c *client) tcpSubscribe(in []string) (err error) {
	var cf node.ConnFeed
	if cf, err = c.argsConnFeed(in); err != nil {
		return
	}
	return c.r.TCP().Subscribe(cf.Address, cf.Feed)
}

func (c *client) tcpUnsubscribe(in []string) (err error) {
	var cf node.ConnFeed
	if cf, err = c.argsConnFeed(in); err != nil {
		return
	}
	return c.r.TCP().Unsubscribe(cf.Address, cf.Feed)
}

func (c *client) tcpAddress(in []string) (err error) {
	if err = c.argsNo(in); err != nil {
		return
	}
	var address string
	if address, err = c.r.TCP().Address(); err != nil {
		return
	}
	if address == "" {
		fmt.Fprintln(out, "  doesn't listen")
		return
	}
	fmt.Fprintln(out, " "+address)
	return
}

//
// udp
//

func (c *client) udpConnect(in []string) (err error) {
	var address string
	if address, err = c.argsAddress(in); err != nil {
		return
	}
	return c.r.UDP().Connect(address)
}

func (c *client) udpDisconnet(in []string) (err error) {
	var address string
	if address, err = c.argsAddress(in); err != nil {
		return
	}
	return c.r.UDP().Disconnect(address)
}

func (c *client) udpSubscribe(in []string) (err error) {
	var cf node.ConnFeed
	if cf, err = c.argsConnFeed(in); err != nil {
		return
	}
	return c.r.UDP().Subscribe(cf.Address, cf.Feed)
}

func (c *client) udpUnsubscribe(in []string) (err error) {
	var cf node.ConnFeed
	if cf, err = c.argsConnFeed(in); err != nil {
		return
	}
	return c.r.UDP().Unsubscribe(cf.Address, cf.Feed)
}

func (c *client) udpAddress(in []string) (err error) {
	if err = c.argsNo(in); err != nil {
		return
	}
	var address string
	if address, err = c.r.UDP().Address(); err != nil {
		return
	}
	if address == "" {
		fmt.Fprintln(out, "  doesn't listen")
		return
	}
	fmt.Fprintln(out, " "+address)
	return
}

//
// connections
//

func printConnections(cs []string) {
	if len(cs) == 0 {
		fmt.Fprintln(out, "  no connections")
		return
	}
	for _, c := range cs {
		fmt.Fprintln(out, " ", c)
	}
}

func (c *client) connections(in []string) (err error) {
	if err = c.argsNo(in); err != nil {
		return
	}
	var cs []string
	if cs, err = c.r.Node().Connections(); err != nil {
		return
	}
	printConnections(cs)
	return
}

func (c *client) connectionsOfFeed(in []string) (err error) {
	var pk cipher.PubKey
	if pk, err = c.argsFeed(in); err != nil {
		return
	}
	var cs []string
	if cs, err = c.r.Node().ConnectionsOfFeed(pk); err != nil {
		return
	}
	printConnections(cs)
	return
}

//
// root objects
//

func (c *client) printRoot(z *registry.Root) {
	var refs = make([]string, 0, len(z.Refs))

	for _, dr := range z.Refs {
		refs = append(refs, dr.Short())
	}

	fmt.Fprintf(out, `  root  %s

    refs:       %#v

	descriptor: %q
	registry:   %s

	feed:       %s
	nonce:      %d
	seq:        %d

	time:       %v

	sig:        %s
	prev:       %s

`,
		z.Hash.Hex(),
		refs,
		string(z.Descriptor),
		z.Reg.String(),
		z.Pub.Hex(),
		z.Nonce,
		z.Seq,
		time.Unix(0, z.Time),
		z.Sig.Hex(),
		z.Prev.Hex(),
	)
}

func (c *client) rootInfo(in []string) (err error) {
	var sl node.RootSelector
	if sl, err = c.argsRoot(in); err != nil {
		return
	}
	var z *registry.Root
	if z, err = c.r.Root().Show(sl.Feed, sl.Nonce, sl.Seq); err != nil {
		return
	}
	c.printRoot(z)
	return
}

func (c *client) rootTree(in []string) (err error) {
	var sl node.RootSelector
	if sl, err = c.argsRoot(in); err != nil {
		return
	}
	var tree string
	if tree, err = c.r.Root().Tree(sl.Feed, sl.Nonce, sl.Seq); err != nil {
		return
	}
	fmt.Fprintln(out, " ", tree)
	return
}

func (c *client) lastRoot(in []string) (err error) {
	var pk cipher.PubKey
	if pk, err = c.argsFeed(in); err != nil {
		return
	}
	var z *registry.Root
	if z, err = c.r.Root().Last(pk); err != nil {
		return
	}
	c.printRoot(z)
	return
}

//
// stat
//

func (c *client) printRootStat(rs skyobject.RootStat) {
	fmt.Fprintln(out, "      hash:", rs.Hash.Hex())
	fmt.Fprintln(out, "      time:", rs.Time)
	fmt.Fprintln(out, "      seq: ", rs.Seq)
}

func round(f float64) (s string) {
	return fmt.Sprintf("%.2f", f)
}

func (c *client) stat(in []string) (err error) {
	if err = c.argsNo(in); err != nil {
		return
	}
	var s *node.Stat
	if s, err = c.r.Node().Stat(); err != nil {
		return
	}

	fmt.Fprintln(out, "  average filling duration:       ", s.Fillavg)

	fmt.Fprintln(out, "  CXDS RPS:                       ", round(s.CXDS.RPS))
	fmt.Fprintln(out, "  CXDS WPS:                       ", round(s.CXDS.WPS))

	fmt.Fprintln(out, "  cache RPS:                      ", round(s.Cache.RPS))
	fmt.Fprintln(out, "  cache WPS:                      ", round(s.Cache.WPS))

	fmt.Fprintln(out, "  average cache cleaning duration:", s.CacheCleaning)

	fmt.Fprintln(out, "  amount of cached objects:       ",
		s.CacheObjects.Amount.String())
	fmt.Fprintln(out, "  volume of cached objects:       ",
		s.CacheObjects.Volume.String())

	fmt.Fprintln(out, "  amount of all objects:          ",
		s.AllObjects.Amount.String())
	fmt.Fprintln(out, "  volume of all objects:          ",
		s.AllObjects.Volume.String())

	fmt.Fprintln(out, "  amount of used objects:         ",
		s.UsedObjects.Amount.String())
	fmt.Fprintln(out, "  volume of used objects:         ",
		s.UsedObjects.Volume.String())

	fmt.Fprintln(out, "  new Root objects per second:    ", s.RootsPerSecond)

	if len(s.Feeds) == 0 {
		fmt.Fprintln(out, "  no feeds")
		return
	}

	for pk, fs := range s.Feeds {
		fmt.Fprintln(out, " ", pk.Hex())

		if len(fs.Heads) == 0 {
			fmt.Fprintln(out, "    no heads")
			continue
		}

		for nonce, hs := range fs.Heads {
			fmt.Fprintln(out, "    ", nonce)

			switch hs.Len {
			case 0:
				fmt.Fprintln(out, "      no root objects")
			case 1:
				fmt.Fprintln(out, "      last Root")
				c.printRootStat(hs.Last)
			default:
				fmt.Fprintln(out, "      first Root")
				c.printRootStat(hs.First)
				fmt.Fprintln(out, "      last Root")
				c.printRootStat(hs.Last)
			}

		}

	}

	return
}

func (c *client) help(in []string) (err error) {
	fmt.Fprint(out, `

  share feed <public key>
    start sharing given feed
  don't share feed <public key>
    stop sharing given feed
  list feeds
    show all feeds the node share

  tcp connect <address>
    connect to tcp address
  tcp disconnect <connection address>
    close tcp connection
  tcp subsribe <connection address> <public key>
    subscribe to feed of peer
  tcp unsubscribe <connection address> <public key>
    unsubscribe from feed of peer
  tcp address
    tcp listening address

  udp connect <address>
    connect to udp address
  udp disconnect <connection address>
    close udp connection
  udp subsribe <connection address> <public key>
    subscribe to feed of peer
  udp unsubscribe <connection address> <public key>
    unsubscribe from feed of peer
  udp address
    udp listening address


  connections
    show all connections
  connections of feed <public key>
    show connections of given feed


  root info <public key> <nonce> <seq>
    show info of selected Root

  root tree <public key> <nonce> <seq>
    print tree of selected Root

  last root <public key>
    show info about last Root of given feed


  stat
    show statistic of node


  help
    show this help messege


  quit or exit
    leave the cli

`)
	return
}

func (c *client) quit([]string) (_ error) {
	fmt.Fprintln(out, "cya")
	return
}

/*

TODO (kostyarin): smart query for Root objects
----------------------------------------------

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


*/
