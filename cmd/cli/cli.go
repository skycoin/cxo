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

	"github.com/skycoin/skycoin/src/mesh/messages"
	"github.com/skycoin/skycoin/src/mesh/node"
	"github.com/skycoin/skycoin/src/mesh/nodemanager"
	"github.com/skycoin/skycoin/src/mesh/transport"

	"github.com/skycoin/cxo/data"
)

const HISTORY = ".cxocli.history"

var (
	ErrUnknowCommand    = errors.New("unknown command")
	ErrMisisngArgument  = errors.New("missing argument")
	ErrTooManyArguments = errors.New("too many arguments")

	commands = []string{
		// mesh related commands
		"add_nodes",
		"list_nodes",
		"connect",
		"list_all_transports",
		"list_transports",
		"build_route",
		"find_route",
		"list_routes",
		// cxo related commands
		"subscribe",
		"unsubscribe",
		"tree",
		"want",
		"got",
		"feeds",
		"stat",
		"terminate",
		"quit",
		"exit",
	}

	status = [...]string{
		0: "DISCONNECTED",
		1: "CONNECTED",
	}
)

func main() {
	var (
		address string
		execute string

		rpc *nodemanager.RPCClient
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
		address = ":" + nodemanager.DEFAULT_PORT
	}

	rpc = nodemanager.RunClient(address)
	defer rpc.Client.Close()

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

	line.SetCtrlCAborts(true)

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

func executeCommand(command string, rpc *nodemanager.RPCClient) (terminate bool,
	err error) {

	defer func() {
		if rec := recover(); rec != nil {
			terminate = true
			if err == nil {
				err = fmt.Errorf("panic: ", rec)
			}
		}
	}()

	ss := strings.Fields(command)
	if len(ss) == 0 {
		return
	}
	switch strings.ToLower(ss[0]) {
	// mesh related commands
	case "add_node":
		err = addNode(rpc, ss[1:])
	case "add_nodes":
		err = addNodes(rpc, ss[1:])
	case "list_nodes":
		err = listNodes(rpc)
	case "connect":
		err = connectNodes(rpc, ss[1:])
	case "list_all_transports":
		err = listAllTransports(rpc)
	case "list_transports":
		err = listTransports(rpc, ss[1:])
	case "build_route":
		err = buildRoute(rpc, ss[1:])
	case "find_route":
		err = findRoute(rpc, ss[1:])
	case "list_routes":
		err = listRoutes(rpc, ss[1:])
	// cxo related commants
	case "subscribe":
		err = subscribe(rpc, ss)
	case "unsubscribe":
		err = unsubscribe(rpc, ss)
	case "tree":
		err = tree(rpc, ss)
	case "want":
		err = want(rpc, ss)
	case "got":
		err = got(rpc, ss)
	case "feeds":
		err = feeds(rpc)
	case "stat":
		err = stat(rpc)
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

  add_node
    ...
  add_nodes
    ...
  list_nodes
    ...
  connect
    ...
  list_all_transports
    ...
  list_transports
    ...
  build_route
    ...
  find_route
    ...
  list_routes
    ...
  subscribe <public key>
    subscribe to given feed
  unsubscribe <public key>
    unsubscribe from given feed
  tree <public key>
    print object tree of given root object
  want <public key>
    want returns list of hashes of missing object of given feed
  got <public key>
    got returns list of objects given feed already has with size
  feeds
    obtain list of feeds of the server
  stat
    obtain database statistic
  terminate
    close the server
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

func subscribe(rpc *nodemanager.RPCClient, ss []string) (err error) {
	var public cipher.PubKey
	if public, err = publicKeyArg(ss); err != nil {
		return
	}
	var subscribed bool
	if err = rpc.Client.Call("cxo.Subscribe", public, &subscribed); err == nil {
		if subscribed {
			fmt.Println("  subscribed")
		} else {
			fmt.Println("  already subscribed")
		}
	}
	return
}

func unsubscribe(rpc *nodemanager.RPCClient, ss []string) (err error) {
	var public cipher.PubKey
	if public, err = publicKeyArg(ss); err != nil {
		return
	}
	var unsubscribed bool
	err = rpc.Client.Call("cxo.Unsubscribe", public, &unsubscribed)
	if err == nil {
		if unsubscribed {
			fmt.Println("  unsubscribed")
		} else {
			fmt.Println("  not subscribed")
		}
	}
	return
}

func tree(rpc *nodemanager.RPCClient, ss []string) (err error) {
	var public cipher.PubKey
	if public, err = publicKeyArg(ss); err != nil {
		return
	}
	var tree []byte
	if err = rpc.Client.Call("cxo.Tree", public, &tree); err == nil {
		if len(tree) == 0 {
			fmt.Println("  empty feed")
			return
		}
		fmt.Println(string(tree))
	}
	return
}

func want(rpc *nodemanager.RPCClient, ss []string) (err error) {
	var public cipher.PubKey
	if public, err = publicKeyArg(ss); err != nil {
		return
	}
	var list []cipher.SHA256
	if err = rpc.Client.Call("cxo.Want", public, &list); err == nil {
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

func got(rpc *nodemanager.RPCClient, ss []string) (err error) {
	var public cipher.PubKey
	if public, err = publicKeyArg(ss); err != nil {
		return
	}
	var list map[cipher.SHA256]int
	if err = rpc.Client.Call("cxo.Got", public, &list); err == nil {
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

func feeds(rpc *nodemanager.RPCClient) (err error) {
	var feeds []cipher.PubKey
	if err = rpc.Client.Call("cxo.Feeds", struct{}{}, &feeds); err != nil {
		return
	}
	if len(feeds) == 0 {
		fmt.Println("  no feeds")
		return
	}
	for _, f := range feeds {
		fmt.Println("  +", f.Hex())
	}
	return
}

func stat(rpc *nodemanager.RPCClient) (err error) {
	var stat data.Stat
	if err = rpc.Client.Call("cxo.Stat", struct{}{}, &stat); err != nil {
		return
	}
	fmt.Println("  Total objects:", stat.Total)
	fmt.Println("  Memory:       ", data.HumanMemory(stat.Memory))
	return
}

func term(rpc *nodemanager.RPCClient) (err error) {
	err = rpc.Client.Call("cxo.Terminate", struct{}{}, &struct{}{})
	if err == io.ErrUnexpectedEOF {
		err = nil
	}
	if err == nil {
		fmt.Println("  terminated")
	}
	return
}

// ========================================================================== //
//                           mesh related commands                            //
// ========================================================================== //

func addNode(client *nodemanager.RPCClient, args []string) (err error) {

	var response []byte
	if response, err = client.SendToRPC("AddNode", args); err != nil {
		return
	}

	var nodeId cipher.PubKey
	if err = messages.Deserialize(response, &nodeId); err != nil {
		return
	}

	fmt.Println("  Added node with ID", nodeId.Hex())
	return
}

func addNodes(client *nodemanager.RPCClient, args []string) (err error) {

	if len(args) == 0 {
		err = fmt.Errorf("Point the number of nodes, please")
		return
	}
	var n int
	if n, err = strconv.Atoi(args[0]); err != nil || n < 1 {
		err = fmt.Errorf("Argument should be a number > 0, not %s", args[0])
		return
	}

	var response []byte
	if response, err = client.SendToRPC("AddNodes", args); err != nil {
		return
	}

	var nodes []cipher.PubKey
	if err = messages.Deserialize(response, &nodes); err != nil {
		return
	}

	for i, nodeId := range nodes {
		fmt.Printf("  %d  Added node with ID %s\n", i, nodeId.Hex())
	}
	fmt.Println("")
	return
}

func listNodes(client *nodemanager.RPCClient) (err error) {

	var nodes []cipher.PubKey
	if nodes, err = getNodes(client); err != nil {
		return
	}

	fmt.Printf("  NODES(%d total):\n\n", len(nodes))
	fmt.Println("  Num\tID\n")
	for i, nodeId := range nodes {
		fmt.Printf("  %d\t%s\n", i, nodeId.Hex())
	}
	return
}

func connectNodes(client *nodemanager.RPCClient, args []string) (err error) {
	if len(args) != 2 {
		err = errors.New("There should be 2 nodes to connect")
		return
	}

	var nodes []cipher.PubKey
	if nodes, err = getNodes(client); err != nil {
		return
	}

	n := len(nodes)
	if n < 2 {
		err = fmt.Errorf("Need at least 2 nodes to connect, have %d\n\n", n)
		return
	}

	node0, node1 := args[0], args[1]

	if !testNodes(node0, n) || !testNodes(node1, n) {
		err = errors.New("Skipping connecting nodes due to errors")
		return
	}

	if node0 == node1 {
		err = errors.New("Node can't be connected to itself")
		return
	}

	var response []byte
	if response, err = client.SendToRPC("ConnectNodes", args); err != nil {
		return
	}

	var transports []messages.TransportId
	if err = messages.Deserialize(response, &transports); err != nil {
		return
	}

	if transports[0] == 0 || transports[1] == 0 {
		err = errors.New("Error connecting nodes, probably already connected")
		return
	}

	fmt.Printf("  Transport ID from node %s to %s is %d\n",
		node0, node1, transports[0])
	fmt.Printf("  Transport ID from node %s to %s is %d\n",
		node1, node0, transports[1])
	return
}

func listAllTransports(client *nodemanager.RPCClient) (err error) {
	var response []byte
	response, err = client.SendToRPC("ListAllTransports", []string{})
	if err != nil {
		return
	}
	var transports []transport.TransportInfo
	if err = messages.Deserialize(response, &transports); err != nil {
		return
	}

	var nodes []cipher.PubKey
	if nodes, err = getNodes(client); err != nil {
		return
	}

	fmt.Printf("  TRANSPORTS(%d total):\n\n", len(transports))
	fmt.Println("  Num\tID\t\t\tStatus\t\tNodeFrom\tNodeTo\n")
	for i, transportInfo := range transports {
		fmt.Printf("  %d\t%d\t%s\t%d\t\t%d\n",
			i,
			transportInfo.TransportId,
			status[transportInfo.Status],
			getNodeNumber(transportInfo.NodeFrom, nodes),
			getNodeNumber(transportInfo.NodeTo, nodes),
		)
	}
	return
}

func listTransports(client *nodemanager.RPCClient, args []string) (err error) {

	if len(args) != 1 {
		err = errors.New("Should be 1 argument, the node number")
		return
	}

	var nodes []cipher.PubKey
	if nodes, err = getNodes(client); err != nil {
		return
	}

	nodenum := args[0]
	n := len(nodes)

	if n == 0 {
		err = errors.New("There are no nodes so far, so no transports")
		return
	}

	if !testNodes(nodenum, n) {
		return
	}

	var response []byte
	if response, err = client.SendToRPC("ListTransports", args); err != nil {
		return
	}

	var transports []transport.TransportInfo
	if err = messages.Deserialize(response, &transports); err != nil {
		return
	}

	fmt.Printf("  TRANSPORTS FOR NODE %s (%d total):\n\n",
		nodenum, len(transports))
	fmt.Println("  Num\tID\t\t\tStatus\t\tNodeFrom\tNodeTo\n")
	for i, transportInfo := range transports {
		fmt.Printf("  %d\t%d\t%s\t%d\t\t%d\n",
			i,
			transportInfo.TransportId,
			status[transportInfo.Status],
			getNodeNumber(transportInfo.NodeFrom, nodes),
			getNodeNumber(transportInfo.NodeTo, nodes),
		)
	}
	fmt.Println("")
	return
}

func buildRoute(client *nodemanager.RPCClient, args []string) (err error) {

	if len(args) < 2 {
		err = errors.New("Route must contain 2 or more nodes")
		return
	}

	var nodes []cipher.PubKey
	if nodes, err = getNodes(client); err != nil {
		return
	}

	n := len(nodes)
	if n < 2 {
		err = fmt.Errorf("Need at least 2 nodes to build a route, have %d", n)
		return
	}

	for _, nodenumstr := range args {
		if !testNodes(nodenumstr, n) {
			return
		}
	}

	var response []byte
	if response, err = client.SendToRPC("BuildRoute", args); err != nil {
		return
	}

	var routes []messages.RouteId
	if err = messages.Deserialize(response, &routes); err != nil {
		return
	}

	fmt.Printf("  ROUTES (%d total):\n\n", len(routes))
	fmt.Println("  Num\tID\n\n")
	for i, routeRuleId := range routes {
		fmt.Printf("  %d\t%d\n", i, routeRuleId)
	}
	fmt.Println("")
	return
}

func findRoute(client *nodemanager.RPCClient, args []string) (err error) {

	if len(args) != 2 {
		err = errors.New("Route should be built between 2 nodes")
		return
	}

	var nodes []cipher.PubKey
	if nodes, err = getNodes(client); err != nil {
		return
	}

	n := len(nodes)
	if n < 2 {
		err = fmt.Errorf("Need at least 2 nodes to build a route, have %d", n)
		return
	}

	for _, nodenumstr := range args {
		if !testNodes(nodenumstr, n) {
			return
		}
	}

	var response []byte
	if response, err = client.SendToRPC("FindRoute", args); err != nil {
		return
	}

	var routes []messages.RouteId
	if err = messages.Deserialize(response, &routes); err != nil {
		return
	}

	fmt.Printf("  ROUTES (%d total):\n\n", len(routes))
	fmt.Println("  Num\tID\n\n")
	for i, routeRuleId := range routes {
		fmt.Printf("  %d\t%d\n", i, routeRuleId)
	}
	fmt.Println("")
	return
}

func listRoutes(client *nodemanager.RPCClient, args []string) (err error) {

	if len(args) != 1 {
		err = errors.New("Should be 1 argument, the node number")
		return
	}

	var nodes []cipher.PubKey
	if nodes, err = getNodes(client); err != nil {
		return
	}

	nodenum := args[0]
	n := len(nodes)

	if n == 0 {
		err = errors.New("There are no nodes so far, so no routes")
		return
	}

	if !testNodes(nodenum, n) {
		return
	}

	var response []byte
	if response, err = client.SendToRPC("ListRoutes", args); err != nil {
		return
	}

	var routes []node.RouteRule
	if err = messages.Deserialize(response, &routes); err != nil {
		return
	}

	fmt.Printf("  ROUTES FOR NODE %s (%d total):\n", nodenum, len(routes))
	for i, routeRule := range routes {
		fmt.Printf("\n  ROUTE %d\n\n", i)
		fmt.Println("  Incoming transport\t", routeRule.IncomingTransport)
		fmt.Println("  Outgoing transport\t", routeRule.OutgoingTransport)
		fmt.Println("  Incoming route\t\t", routeRule.IncomingRoute)
		fmt.Println("  Outgoing route\t\t", routeRule.OutgoingRoute)
		fmt.Println("  ------------------")
	}
	fmt.Println("")
	return
}

//=============helper functions===========

func getNodes(client *nodemanager.RPCClient) ([]cipher.PubKey, error) {
	response, err := client.SendToRPC("ListNodes", []string{})
	if err != nil {
		return []cipher.PubKey{}, err
	}

	var nodes []cipher.PubKey
	err = messages.Deserialize(response, &nodes)
	if err != nil {
		return []cipher.PubKey{}, err
	}
	return nodes, nil
}

func getNodeNumber(nodeIdToFind cipher.PubKey, nodes []cipher.PubKey) int {
	for i, nodeId := range nodes {
		if nodeIdToFind == nodeId {
			return i
		}
	}
	return -1
}

func testNodes(node string, n int) bool {

	nodeNumber, err := strconv.Atoi(node)
	if err == nil {
		if nodeNumber >= 0 && nodeNumber < n {
			return true
		}
	}

	fmt.Printf("\nNode %s should be a number from 0 to %d\n", node, n-1)
	return false
}
