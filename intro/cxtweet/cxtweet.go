package main

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/logrusorgru/aurora"
	"github.com/peterh/liner"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/data"
	"github.com/skycoin/cxo/node"
	cxo "github.com/skycoin/cxo/skyobject"
)

const (
	Bind   string = "[::]:0"
	Colors bool   = true
)

var (
	commands = []string{
		"new tweet ", // emit new tweet

		"show tweet ", // show last tweet, or last n tweets
		"show user ",  // show user info
		"show feed ",  // show all tweets

		"live ", // show new tweet in real time
		"hide ", // don't show real-time tweets

		"connect ",    // address
		"disconnect ", // address

		"subscribe ",   // feed
		"unsubscribe ", // feed

		"connections ", // connections
		"feeds ",       // feeds

		"address ", // listening address

		"help ", // get help

		"quit", // exit
		"exit", // quit
	}

	au   aurora.Aurora // ANSI-colors
	s    *node.Node    // node instance
	line *liner.State  // ask for some strings

	liveView = struct {
		sync.Mutex // async  access
		last       map[cipher.PubKey]liveFeed
	}{
		last: make(map[cipher.PubKey]liveFeed),
	}

	thisUser struct {
		sk   cipher.SecKey
		pk   cipher.PubKey
		name string

		pack *cxo.Pack // user's feed
	}
)

type liveFeed struct {
	live      bool  // enable live view
	lastShown int64 // time of last shown tweet
}

func main() {

	var flags struct {
		bind   string
		colors bool
		name   string
		sk     string
	}

	flag.StringVar(&flags.bind,
		"b",
		Bind,
		"listening addres")
	flag.BoolVar(&flags.colors,
		"c",
		Colors,
		"ANSI colors")
	flag.StringVar(&flags.name,
		"new",
		"",
		"name for new user")
	flag.StringVar(&flags.sk,
		"load",
		"",
		"load user by secret key")

	flag.Parse()

	au = aurora.NewAurora(flags.colors)

	if flags.name != "" {
		if flags.sk != "" {
			fmt.Println("new or load?")
			return
		}
		thisUser.name = flags.name
	} else if flags.sk != "" {
		sk, err := cipher.SecKeyFromHex(flags.sk)
		if err != nil {
			fmt.Println(err)
			return
		}
		thisUser.sk = sk
		thisUser.pk = cipher.PubKeyFromSecKey(sk)
	} else {
		fmt.Println("pass 'new' or 'load' flag")
		return
	}

	// register types
	reg := cxo.NewRegistry(func(r *cxo.Reg) {
		r.Register("tw.User", User{})
		r.Register("tw.Tweet", Tweet{})
		r.Register("tw.Feed", Feed{})
	})

	nc := node.NewConfig()

	nc.Skyobject.Registry = reg // registry

	nc.EnableRPC = false   // disable RPC
	nc.Listen = flags.bind // listen
	nc.PingInterval = 0    // disabel pings

	nc.InMemoryDB = true   // use DB in memeory
	nc.DataDir = ""        // don't initialize defautl ~/.skycoin/cxo dir
	nc.PublicServer = true // allow request list of feeds

	nc.OnCreateConnection = func(c *node.Conn) {
		fmt.Println("new conenction:", c.Address())
	}

	nc.OnCloseConnection = func(c *node.Conn) {
		fmt.Println("connection closed:", c.Address())
	}

	// accept all incoming subscription
	nc.OnSubscribeRemote = func(c *node.Conn, pk cipher.PubKey) (reject error) {
		if err := c.Node().AddFeed(pk); err != nil {
			fmt.Println(au.Red("[CRIT] replace you HDD"))
		}
		return
	}
	nc.OnRootFilled = liveCallback // show live tweets

	var err error
	if s, err = node.NewNode(nc); err != nil {
		fmt.Println(au.Red("[CRIT]"), err)
		return
	}
	defer s.Close()

	// load user
	if flags.name != "" {
		// create user
		err = createUser(flags.name)
	} else {
		// load user from DB
		err = loadUser()
	}
	if err != nil {
		fmt.Println(au.Red("[ERR]"), err)
		return
	}

	line = liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(false)

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range commands {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	var cmd string
	var terminate bool
	fmt.Println("enter 'help' to get help")
	for {
		cmd, err = line.Prompt("> ")
		if err != nil && err != liner.ErrPromptAborted {
			fmt.Println(au.Red("[FATAL]"), err)
			return
		}
		terminate, err = executeCommand(cmd)
		if err != nil {
			if terminate {
				fmt.Println(au.Red("[CRIT]"), err)
			} else {
				fmt.Println(au.Red("[ERR]"), err)
			}
		}
		if terminate {
			return
		}
		line.AppendHistory(cmd)
	}

}

func connDir(incoming bool) string {
	if incoming {
		return "<--"
	}
	return "-->"
}

func createUser(name string) (err error) {
	thisUser.pk, thisUser.sk = cipher.GenerateKeyPair()

	cnt := s.Container() // this call is cheap

	if err = cnt.AddFeed(thisUser.pk); err != nil {
		return
	}

	// create empty feed

	var pack *cxo.Pack
	pack, err = cnt.NewRoot(thisUser.pk,
		thisUser.sk,
		0,
		cnt.CoreRegistry().Types())
	if err != nil {
		return
	}
	// defer pack.Close() <== we will keep this pack up to closing
	thisUser.pack = pack

	pack.Append(Feed{
		User: pack.Ref(User{name}),
	})

	if err = pack.Save(); err != nil {
		return
	}
	// don't publish

	fmt.Printf("user %q has been created\n", name)
	fmt.Println("  public key:", thisUser.pk.Hex())
	fmt.Println("  secret key:", thisUser.sk.Hex())

	return
}

func loadUser() (err error) {
	cnt := s.Container()

	var r *cxo.Root
	if r, err = cnt.LastRoot(thisUser.pk); err != nil {
		return
	}
	defer cnt.UnholdRoot(r) // unhold it then

	var pack *cxo.Pack
	pack, err = cnt.Unpack(r,
		cxo.ViewOnly,
		cnt.CoreRegistry().Types(),
		thisUser.sk)
	if err != nil {
		return
	}
	// defer pack.Close() <== we will keep this pack up to closing

	thisUser.pack = pack
	return
}

func getUser(feed *Feed) (usr *User, err error) {
	var usrInterface interface{}
	if usrInterface, err = feed.User.Value(); err != nil {
		return
	}
	usr = usrInterface.(*User)
	return
}

func getFeed(pack *cxo.Pack) (feed *Feed, err error) {
	var feedInterface interface{}
	if feedInterface, err = pack.RefByIndex(0); err != nil {
		return
	}
	if feedInterface == nil {
		return nil, errors.New("feed is nil")
	}
	feed = feedInterface.(*Feed)
	return
}

func getLastShown(pk cipher.PubKey) (lastShown int64, live bool) {
	liveView.Lock()
	defer liveView.Unlock()

	lf := liveView.last[pk]
	return lf.lastShown, lf.live
}

func setLastShown(pk cipher.PubKey, lastShown int64) {
	liveView.Lock()
	defer liveView.Unlock()

	lf := liveView.last[pk]  // get
	lf.lastShown = lastShown // update
	liveView.last[pk] = lf   // set
}

func setLive(pk cipher.PubKey, live bool) {
	liveView.Lock()
	defer liveView.Unlock()

	lf := liveView.last[pk] // get
	if true == live {
		lf.lastShown = time.Now().UnixNano() // current time point
	}
	lf.live = live         // update
	liveView.last[pk] = lf // set
}

func printTweet(usr *User, tweet *Tweet) {

	tp := time.Unix(0, tweet.Time)

	fmt.Println(au.Green("----"))
	fmt.Println(au.Cyan(usr.Name).Bold(), au.Gray(tp.Format(time.Kitchen)))
	fmt.Println("")

	fmt.Println(au.Brown("  " + tweet.Head).Bold())
	fmt.Println(" ", tweet.Body)

	fmt.Println("")
	fmt.Println(au.Green("----"))
}

// show live tweets
func liveCallback(c *node.Conn, root *cxo.Root) {

	// live
	lastShown, ok := getLastShown(root.Pub) // root.Pub is feed (public key)
	if !ok {
		return
	}

	// get underlying "container"
	so := c.Node().Container()

	// unpack the Root to lookup it

	pack, err := so.Unpack(root, // the Root
		cxo.ViewOnly,              // view only
		so.CoreRegistry().Types(), // registered types
		cipher.SecKey{})           // doesn't need

	if err != nil {
		fmt.Println(au.Red("[ERR] error unpacking Root:"), err)
		return
	}
	defer pack.Close()

	// iterate from tail to find last shown
	var stack []*Tweet
	var user *User

	err = descendTweets(pack, func(usr *User, _ int, tweet *Tweet) error {
		// for all tweets of a feed the user is the same
		user = usr // keep

		if tweet.Time > lastShown {
			stack = append(stack, tweet)
		} else {
			return cxo.ErrStopIteration // break
		}
		return nil // continue
	})

	if err != nil {
		fmt.Println(au.Red("[ERR] can't get tweets:"), err)
		return
	}

	// print new tweets
	for i := len(stack) - 1; i >= 0; i-- {
		printTweet(user, stack[i])
	}

	if len(stack) > 0 {
		// the satck[0] is first from tail (i.e. last) tweet
		setLastShown(root.Pub, stack[0].Time)
	}

}

// print last n tweets of feed
func printLastTweets(feed cipher.PubKey, n int) {

	if n <= 0 {
		return
	}

	cnt := s.Container()
	root, err := cnt.LastRoot(feed)

	if err != nil {
		if err == data.ErrNotFound {
			fmt.Println("(empty feed)")
			return
		}
		fmt.Println(au.Red("[ERR] can't access feed:"), err)
		return
	}

	pack, err := cnt.Unpack(root, // root
		cxo.ViewOnly,               // view only
		cnt.CoreRegistry().Types(), // registered types to decode
		cipher.SecKey{})            // view only (no secret key needed)

	if err != nil {
		fmt.Println(au.Red("[ERR] error unpacking Root:"), err)
		return
	}

	err = descendTweets(pack, func(usr *User, _ int, tweet *Tweet) error {
		printTweet(usr, tweet)
		n--
		if n <= 0 {
			setLastShown(feed, tweet.Time) // last shown
			return cxo.ErrStopIteration
		}
		return nil
	})
	if err != nil {
		fmt.Println(au.Red("[ERR]"), err)
	}

}

func descendTweets(pack *cxo.Pack, df func(*User, int, *Tweet) error) error {
	feed, err := getFeed(pack)
	if err != nil {
		return err
	}
	// get user info
	userInterface, err := feed.User.Value()
	if err != nil {
		return err
	}
	user := userInterface.(*User) // got it
	return feed.Tweets.Descend(func(i int, el *cxo.RefsElem) (err error) {
		var tweetInterface interface{}
		if tweetInterface, err = el.Value(); err != nil {
			return
		}
		tweet := tweetInterface.(*Tweet)

		return df(user, i, tweet)
	})
}

func args(s string, cmd string) (ss []string) {
	return strings.Fields(strings.TrimPrefix(s, cmd))
}

func executeCommand(s string) (term bool, err error) {
	switch {

	case strings.HasPrefix(s, "new tweet"):
		if err = newTweet(); err != nil {
			term = true
		}

	case strings.HasPrefix(s, "show tweet"):
		err = showTweet(args(s, "show tweet"))
	case strings.HasPrefix(s, "show user"):
		err = showUser(args(s, "show user"))
	case strings.HasPrefix(s, "show feed"):
		err = showFeed(args(s, "show feed"))

	case strings.HasPrefix(s, "live"):
		err = live(args(s, "live"))
	case strings.HasPrefix(s, "hide"):
		err = hide(args(s, "hide"))

	case strings.HasPrefix(s, "connections"): // connetions before connect
		connections()
	case strings.HasPrefix(s, "feeds"):
		feeds()
	case strings.HasPrefix(s, "address"):
		address()

	case strings.HasPrefix(s, "connect"):
		err = connect(args(s, "connect"))
	case strings.HasPrefix(s, "disconnect"):
		err = disconnect(args(s, "disconnect"))

	case strings.HasPrefix(s, "subscribe"):
		err = subscribe(args(s, "subscribe"))
	case strings.HasPrefix(s, "unsubscribe"):
		err = unsubscribe(args(s, "unsubscribe"))

	case strings.HasPrefix(s, "help"):
		help()

	case strings.HasPrefix(s, "quit"):
		fallthrough
	case strings.HasPrefix(s, "exit"):
		fmt.Println("cya")
		term = true

	default:
		if s == "" {
			return
		}
		return false, errors.New("unknown commend: " + s)
	}
	return
}

func askForString(name string) (val string, err error) {
	fmt.Println("·· enter", name)
	val, err = line.Prompt("··> ")
	if err != nil && err != liner.ErrPromptAborted {
		fmt.Println(au.Red("[FATAL]"), err)
	}
	return
}

func newTweet() (err error) {
	var header, body string
	if header, err = askForString("header"); err != nil {
		return
	}
	if body, err = askForString("body"); err != nil {
		return
	}

	var feed *Feed
	if feed, err = getFeed(thisUser.pack); err != nil {
		return
	}

	feed.Tweets.Append(&Tweet{
		Time: time.Now().UnixNano(),
		Head: header,
		Body: body,
	})
	if err = thisUser.pack.SetRefByIndex(0, feed); err != nil {
		return
	}
	if err = thisUser.pack.Save(); err != nil {
		return
	}
	s.Publish(thisUser.pack.Root()) // publish cahgnes
	return
}

func argsRange(args []string, f, t int) (err error) {
	if ln := len(args); ln < f {
		return errors.New("too few arguments")
	} else if ln > t {
		return errors.New("too many arguments")
	}
	return
}

func argsPkRange(args []string, f, t int) (pk cipher.PubKey, err error) {
	if err = argsRange(args, 1, 2); err != nil {
		return
	}
	return cipher.PubKeyFromHex(args[0])
}

func showTweet(args []string) (err error) {
	// pk [n]
	var pk cipher.PubKey
	if pk, err = argsPkRange(args, 1, 2); err != nil {
		return
	}
	var n int
	if len(args) == 2 {
		var i64 int64
		if i64, err = strconv.ParseInt(args[1], 64, 64); err != nil {
			return
		}
		n = int(i64)
	}

	var pack *cxo.Pack
	if pack, err = getPack(pk); err != nil {
		return
	}
	defer pack.Close()

	err = descendTweets(pack, func(usr *User, _ int, tw *Tweet) (err error) {
		printTweet(usr, tw)
		if n <= 0 {
			return cxo.ErrStopIteration
		}
		return
	})
	return
}

func printUser(usr *User) {
	fmt.Println(au.Gray("User:"), au.Gray(usr.Name).Bold())
}

// view only
func getPack(pk cipher.PubKey) (pack *cxo.Pack, err error) {
	cnt := s.Container()

	var r *cxo.Root
	if r, err = cnt.LastRoot(pk); err != nil {
		return
	}
	defer cnt.UnholdRoot(r)

	return cnt.Unpack(r,
		cxo.ViewOnly,
		cnt.CoreRegistry().Types(),
		cipher.SecKey{})
}

func getUserByFeed(feed *Feed) (usr *User, err error) {
	var usrInterface interface{}
	if usrInterface, err = feed.User.Value(); err != nil {
		return
	}
	if usr = usrInterface.(*User); usr == nil {
		err = errors.New("nil user")
	}
	return
}

func showUser(args []string) error {
	var pack *cxo.Pack
	if len(args) == 0 {
		pack = thisUser.pack
	} else {
		if pk, err := argsPkRange(args, 1, 1); err != nil {
			return err
		} else if pack, err = getPack(pk); err != nil {
			return err
		}
		defer pack.Close() // close to unhold underlying root
	}
	if feed, err := getFeed(pack); err != nil {
		return err
	} else if usr, err := getUserByFeed(feed); err != nil {
		return err
	} else {
		printUser(usr)
	}
	return nil
}

func getTweet(el *cxo.RefsElem) (tw *Tweet, err error) {
	var twInterface interface{}
	if twInterface, err = el.Value(); err != nil {
		return
	}
	tw = twInterface.(*Tweet)
	return
}

func printFeed(feed *Feed) {
	fmt.Println("---------------")
	if ln, err := feed.Tweets.Len(); err != nil {
		fmt.Println(au.Red("[ERR]"), err)
	} else if ln == 0 {
		fmt.Println("(empty feed)")
	} else {
		usr, err := getUserByFeed(feed)
		if err != nil {
			fmt.Println(au.Red("[ERR]"), err)
			return
		}
		err = feed.Tweets.Ascend(func(i int, el *cxo.RefsElem) error {
			if tw, err := getTweet(el); err != nil {
				return err
			} else {
				printTweet(usr, tw)
			}
			return nil
		})
		if err != nil {
			fmt.Println(au.Red("[ERR]"), err)
		}
	}
	fmt.Println("---------------")
}

func showFeed(args []string) error {
	var pack *cxo.Pack
	if len(args) == 0 {
		pack = thisUser.pack // current
	} else {
		if pk, err := argsPkRange(args, 1, 1); err != nil {
			return err
		} else if pack, err = getPack(pk); err != nil {
			return err
		}
		defer pack.Close() // close to unhold underlying root
	}
	if feed, err := getFeed(pack); err != nil {
		return err
	} else {
		printFeed(feed)
	}
	return nil
}

func live(args []string) error {
	if len(args) == 0 {
		// show all live feeds
		liveView.Lock()
		defer liveView.Unlock()
		for pk := range liveView.last {
			fmt.Println("  -", pk.Hex())
		}
		return nil
	}
	for _, s := range args {
		pk, err := cipher.PubKeyFromHex(s)
		if err != nil {
			return err
		}
		setLive(pk, true)
	}
	return nil
}

func hide(args []string) error {
	for _, s := range args {
		pk, err := cipher.PubKeyFromHex(s)
		if err != nil {
			return err
		}
		setLive(pk, false)
	}
	return nil
}

func connect(args []string) (err error) {
	if err = argsRange(args, 1, 1); err != nil {
		return
	}
	return s.Connect(args[0])
}

func disconnect(args []string) (err error) {
	if err = argsRange(args, 1, 1); err != nil {
		return
	}
	if c := s.Connection(args[0]); c != nil {
		return c.Close()
	}
	return
}

func subscribe(args []string) error {
	for _, x := range args {
		if pk, err := cipher.PubKeyFromHex(x); err != nil {
			return err
		} else {
			if err = s.AddFeed(pk); err != nil {
				return err
			}
			for _, c := range s.Connections() {
				if err := c.Subscribe(pk); err != nil {
					fmt.Println(au.Red("[ERR]"), "can't subscribe to",
						c.Address()+":", err)
				}
			}
		}
	}
	return nil
}

func unsubscribe(args []string) error {
	for _, x := range args {
		if pk, err := cipher.PubKeyFromHex(x); err != nil {
			return err
		} else {
			return s.DelFeed(pk)
		}
	}
	return nil
}

func connections() {
	for _, c := range s.Connections() {
		fmt.Printf("  %s %s\n",
			au.Cyan(connDir(c.Gnet().IsIncoming())),
			c.Address())
	}
}

func feeds() {
	for _, f := range s.Feeds() {
		fmt.Println(" ", f.Hex())
	}
}

func address() {
	fmt.Println(" ", s.Pool().Address())
}

func help() {
	fmt.Println(`
    new tweet
        emit new tweet

    show tweet <pk> [n]
        show last n tweets of given feed
    show user <pk>
    	show user of given feed
    show feed <pk>
    	show all tweets of given feed

    live [pk]
        show new tweet of given feed in real time,
        wihtout arguments it shows all live feeds
    hide <pk>
        don't show real-time tweets of given feed

    connect <address>
        connect to given address
    disconnect <address>
        disconnect from given address

    subscribe <pk>
        unsubscribe to given feed (it can be space sperated list of feeds)
    unsubscribe <pk>
        unsubscribe from given feed (it can be space sperated list of feeds)

    connections
        list all connections
    feeds
       list all known feeds

    help
       show this help message

    quit
        exit
    exit
        quit
`)
}
