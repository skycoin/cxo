package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	//"time"

	"github.com/peterh/liner"
)

const (
	DEFAULT_CXOD_ADDR = "http://127.0.0.1:6481"
	DEFAULT_TIMEOUT   = 0

	HISTORY_FILE   = ".cxo_cli_history"
	LOG_PREFIX     = "[cxo cli] "
	ERROR_BODY_LEN = 500
)

// - list subscriptions (number of peers, size of data for subscription)
// - add subscription
// - list connections for subscription
// - add connection (IP:port) for subscription
// - remove subscription
// - get data size for subscriptions

func main() {
	// initialize logger
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[cxo cli] ")

	// flags
	addr := flag.String("a", DEFAULT_CXOD_ADDR, "server address")
	timeout := flag.Duration("t", DEFAULT_TIMEOUT, "request/response timeout")
	help := flag.Bool("h", false, "show help")
	debug := flag.Bool("d", false, "print debug logs")
	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}

	// http client
	client := Client{
		addr: *addr,
		Client: http.Client{
			Timeout: *timeout,
		},
		debug: *debug,
	}

	// liner
	line := liner.NewLiner()
	defer line.Close()

	readHistory(line)
	defer storeHistory(line)

	line.SetCtrlCAborts(true)
	line.SetCompleter(autoComplite)
	line.SetTabCompletionStyle(liner.TabPrints)

	log.Print("starting client")
	log.Print("address:    ", *addr)
	if *timeout == 0 {
		log.Print("timeout:    no limits")
	} else {
		log.Print("timeout:    ", *timeout)
	}
	log.Print("debug logs: ", *debug)

	fmt.Println("enter 'help' to get help")
	var inpt string
	var err error
	// prompt loop
	for {
		inpt, err = line.Prompt("> ")
		if err != nil {
			log.Print("fatal: ", err)
			return
		}
		inpt = strings.TrimSpace(strings.ToLower(inpt))
		switch {

		case strings.HasPrefix(inpt, "list subscriptions"):
			client.listSubscriptions(trim(inpt, "list subscriptions"))

		case strings.HasPrefix(inpt, "list nodes"):
			client.listNodes()

		case strings.HasPrefix(inpt, "list"):
			fmt.Println(`list what?
	- list subscriptions
	- list nodes`)
			continue

		case strings.HasPrefix(inpt, "add subscription"):
			client.addSubscription(trim(inpt, "add subscription"))

		case strings.HasPrefix(inpt, "add node"):
			client.addNode(trim(inpt, "add node"))

		case strings.HasPrefix(inpt, "add"):
			fmt.Println(`add what?
	- add subscription
	- add node`)
			continue

		case strings.HasPrefix(inpt, "remove subscription"):
			niy()

		case strings.HasPrefix(inpt, "data size"):
			niy()

		case strings.HasPrefix(inpt, "help"):
			printHelp()

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

// historyFilePath returns path to ~/HISTORY_FILE or error if any
func historyFilePath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, HISTORY_FILE), nil
}

// readHistory from history file
func readHistory(line *liner.State) {
	// don't report errors
	pth, err := historyFilePath()
	if err != nil {
		return
	}
	if fl, err := os.Open(pth); err == nil {
		line.ReadHistory(fl)
		fl.Close()
	}
}

// storeHistory to history file
func storeHistory(line *liner.State) {
	pth, err := historyFilePath()
	if err != nil {
		log.Print("error obtaining history file path: ", err)
		return
	}
	fl, err := os.Create(pth)
	if err != nil {
		log.Print("error creating histrory file: ", err)
		return
	}
	defer fl.Close()
	line.WriteHistory(fl)
}

var complets = []string{
	"list subscriptions ",
	"list nodes ",
	"list ",
	"add subscription ",
	"add node ",
	"remove subscription ",
	"data size ",
	"help ",
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

// TODO: help
func printHelp() {
	fmt.Print(`Available commands:

	list subscriptions <node id>
		list subscriptions (number of peers, size of data for subscription)
	list nodes
		list nodes
	add subscription <node id> <host:port> <pubKey>
		add subscription
	add node <secKey>
		add node by its secret key
	remove subscription
		remove subscription (not implemented)
	data size
		get data size (not implemented)
	help
		show this help message
	exit or
	quit
		quit cli
`)
}

func niy() {
	fmt.Println("NOT IMPLEMENTED YET")
}

// time cmd prefix from inpt and time spaces
func trim(inpt string, cmd string) string {
	return strings.TrimSpace(strings.TrimPrefix(inpt, cmd))

}

// request/reply functions

// net/http.Client wrapper
type Client struct {
	http.Client
	addr  string
	debug bool
}

func (c *Client) Debug(args ...interface{}) {
	if c.debug {
		log.Print(args...)
	}
}

//
// list
//

func (c *Client) listSubscriptions(nodeId string) {
	if nodeId == "" {
		fmt.Println("node id required: list subscriptions <node id>")
		return
	}
	// sanitize nodeId
	nodeId = url.QueryEscape(nodeId)
	// request path
	req := c.addr + "/manager/nodes/" + nodeId + "/subscriptions"
	c.Debug("[GET] ", req)
	// obtain
	resp, err := c.Get(req)
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	c.Debug("response status: ", resp.Status)
	defer resp.Body.Close()
	// error returns JSONResponse
	if resp.StatusCode != 200 {
		// decode JSON-response
		jr, err := readResponse(resp)
		if err != nil {
			fmt.Println("error reading response: ", err)
			return
		}
		fmt.Println("response error:", jr.Detail)
		return
	}
	// read list
	li, err := readList(resp)
	if err != nil {
		fmt.Println("error reading response: ", err)
		return
	}
	// huminize the list
	if len(li) == 0 {
		fmt.Println("there aren't subscriptions")
		return
	}
	for _, item := range li {
		fmt.Println(fmt.Sprintf("%s:%d %s", item.IP, item.Port, item.PubKey))
	}
}

func (c *Client) listNodes() {
	// request path
	req := c.addr + "/manager/nodes/"
	c.Debug("[GET] ", req)
	// obtain
	resp, err := c.Get(req)
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	c.Debug("response status: ", resp.Status)
	defer resp.Body.Close()
	// error returns JSONResponse
	if resp.StatusCode != 200 {
		// decode JSON-response
		jr, err := readResponse(resp)
		if err != nil {
			fmt.Println("error reading response: ", err)
			return
		}
		fmt.Println("response error:", jr.Detail)
		return
	}
	// read list
	li, err := readList(resp)
	if err != nil {
		fmt.Println("error reading response: ", err)
		return
	}
	// huminize the list
	if len(li) == 0 {
		fmt.Println("  there aren't nodes")
		return
	}
	for _, item := range li {
		fmt.Println(fmt.Sprintf("  %s:%d %s", item.IP, item.Port, item.PubKey))
	}
}

//
// add
//

func (c *Client) addSubscription(args string) {
	// POST "/manager/nodes/:node_id/subscriptions"
	//   {"ip": "host:port", "pubKey": "theKey"}
	var reqp, reqb string // requset URL and request body
	switch ss := strings.Fields(args); len(ss) {
	case 0, 1, 2:
		fmt.Println("to few arguments, want <node id>, <host:port> <pub key>")
		return
	case 3:
		reqp = c.addr + "/manager/nodes/" + url.QueryEscape(ss[0]) +
			"/subscriptions"
		reqb = fmt.Sprintf(`{"ip":%q,"pubKey":%q}`, ss[1], ss[2])
	default:
		fmt.Println("to many arguments, want <node id>, <host:port> <pub key>")
		return
	}
	//
	c.Debug("[POST] ", reqp, reqb)
	resp, err := c.Post(reqp, "application/json", strings.NewReader(reqb))
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	c.Debug("response status: ", resp.Status)
	defer resp.Body.Close()
	// anyway it's JSONResponse
	jr, err := readResponse(resp)
	if err != nil {
		fmt.Println("error reading response: ", err)
		return
	}
	// detailed error or success message
	fmt.Println(" ", jr.Detail)
}

func (c *Client) addNode(secKey string) {
	// POST "/manager/nodes"
	//   {"secKey": "theKey"}

	// requset URL and request body
	var reqp, reqb string = c.addr + "/manager/nodes",
		fmt.Sprintf(`{"secKey":%q}`, secKey)

	//
	c.Debug("[POST] ", reqp, reqb)
	resp, err := c.Post(reqp, "application/json", strings.NewReader(reqb))
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	c.Debug("response status: ", resp.Status)
	defer resp.Body.Close()
	// anyway it's JSONResponse
	jr, err := readResponse(resp)
	if err != nil {
		fmt.Println("error reading response: ", err)
		return
	}
	// detailed error or success message
	fmt.Println(" ", jr.Detail)
}

// helpers

func readResponse(resp *http.Response) (jr JSONResponse, err error) {
	err = json.NewDecoder(resp.Body).Decode(&jr)
	return
}

func readList(resp *http.Response) (li []Item, err error) {
	err = json.NewDecoder(resp.Body).Decode(&li)
	return
}

// nessesary JSON-structures

// skycoin/cxo/gui/errors.go
type JSONResponse struct {
	Code   string                  `json:"code,omitempty"`
	Status int                     `json:"status,omitempty"`
	Detail string                  `json:"detail,omitempty"`
	Meta   *map[string]interface{} `json:"meta,omitempty"`
}

// list nodes or list subscriptions
type Item struct {
	IP     string `json:"ip"`
	PubKey string `json:"pubKey"`
	Port   int    `json:"port"`
}
