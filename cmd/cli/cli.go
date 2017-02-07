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

		case strings.HasPrefix(inpt, "list connections"):
			client.listConnections()

		case strings.HasPrefix(inpt, "list"):
			fmt.Println(`list what?
	- list subscriptions
	- list connections`)
			continue

		case strings.HasPrefix(inpt, "add subscription"):
			client.addSubscription(trim(inpt, "add subscription"))

		case strings.HasPrefix(inpt, "add connection"):
			client.addConnection(trim(inpt, "add connection"))

		case strings.HasPrefix(inpt, "add"):
			fmt.Println(`add what?
	- add subscription
	- add connection`)
			continue

		case strings.HasPrefix(inpt, "remove subscription"):
			client.removeSubscription(trim(inpt, "remove subscription"))

		case strings.HasPrefix(inpt, "stat"):
			client.getStat()

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
	"list connections ",
	"list ",
	"add subscription ",
	"add connection ",
	"remove subscription ",
	"stat ",
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

	list subscriptions <connection id>
		list subscriptions (number of peers, size of data for subscription)
	list connections
		list conections
	add subscription <connection id> <host:port> <pubKey>
		add subscription
	add connection <secKey>
		add connection by its secret key
	remove subscription <cnnetion id> <pubKey>
		remove subscription
	stat
		get statistic (total objects, memory)
	help
		show this help message
	exit or
	quit
		quit cli
`)
}

// trim cmd prefix from inpt and trim spaces
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

func (c *Client) listSubscriptions(connId string) {
	if connId == "" {
		fmt.Println(
			"connection id required: list subscriptions <connection id>")
		return
	}
	// sanitize connId
	connId = url.QueryEscape(connId)
	// request path
	req := c.addr + "/manager/nodes/" + connId + "/subscriptions"
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

func (c *Client) listConnections() {
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
		fmt.Println(
			"to few arguments, want <connection id>, <host:port> <pub key>")
		return
	case 3:
		reqp = c.addr + "/manager/nodes/" + url.QueryEscape(ss[0]) +
			"/subscriptions"
		reqb = fmt.Sprintf(`{"ip":%q,"pubKey":%q}`, ss[1], ss[2])
	default:
		fmt.Println(
			"to many arguments, want <connection id>, <host:port> <pub key>")
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

func (c *Client) addConnection(secKey string) {
	// POST "/manager/nodes"
	//   {"secKey": "theKey"}

	if secKey == "" {
		fmt.Println("secKey required: add connection <secKey>")
		return
	}

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

//
// remove subscription
//

func (c *Client) removeSubscription(args string) {
	// DELETE "/manager/nodes/:node_id/subscriptions/:subscription_id"

	var reqs string
	switch ss := strings.Fields(args); len(ss) {
	case 0, 1:
		fmt.Println("to few argumets, want: <connection id> <subscription id>")
		return
	case 2:
		reqs = c.addr + "/manager/nodes/" + url.QueryEscape(ss[0]) +
			"/subscriptions/" + url.QueryEscape(ss[1])
	default:
		fmt.Println("to many argumets, want: <connection id> <subscription id>")
		return
	}

	req, err := http.NewRequest("DELETE", reqs, nil)
	if err != nil {
		log.Print("request creating error:", err) // BUG
		return
	}

	c.Debug("[DELETE] ", reqs)
	resp, err := c.Do(req)
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

//
// stat
//

func (c *Client) getStat() {
	// GET "/object1/_stat"

	var req string = c.addr + "/object1/_stat"
	c.Debug("[GET] ", req)
	resp, err := c.Get(req)
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	c.Debug("response status: ", resp.Status)
	defer resp.Body.Close()
	// no error descripto
	if resp.StatusCode != 200 {
		fmt.Println("response error:", resp.Status)
		return
	}
	// read stat
	var stat Statistic
	if err = json.NewDecoder(resp.Body).Decode(&stat); err != nil {
		fmt.Println("error decoding response:", err)
		return
	}
	// print the stat
	fmt.Println("total objects:", stat.Total)
	fmt.Println("memory:       ", humanMemory(stat.Memory))
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

func humanMemory(bytes int) string {
	var fb float64 = float64(bytes)
	var ms string = "B"
	for _, m := range []string{"KiB", "MiB", "GiB"} {
		if fb > 1024.0 {
			fb = fb / 1024.0
			ms = m
			continue
		}
		break
	}
	return fmt.Sprintf("%.2f%s", fb, ms)
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

// stat cxo/data/db.go
type Statistic struct {
	Total  int `json:"total"`
	Memory int `json:"memory"`
}
