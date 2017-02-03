package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	//"net/url"
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
	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}

	// http client
	client := Client{
		Addr: *addr,
		Client: http.Client{
			Timeout: *timeout,
		},
	}

	// liner
	line := liner.NewLiner()
	defer line.Close()

	readHistory(line)
	defer storeHistory(line)

	line.SetCtrlCAborts(true)
	line.SetCompleter(autoComplite)

	log.Print("starting client")
	log.Print("address: ", *addr)
	if *timeout == 0 {
		log.Print("timeout: no limits")
	} else {
		log.Print("timeout: ", *timeout)
	}

	fmt.Print("enter 'help' to get help")
	var inpt string
	var err error
	// prompt loop
	for {
		inpt, err = line.Prompt("> ")
		if err != nil {
			log.Print("fatal: ", err)
			return
		}
		inpt = strings.ToLower(inpt)
		switch {
		case strings.HasPrefix(inpt, "list subscriptions"):
			err = listSubscriptions(&client, trim(inpt, "list subscriptions"))
		case strings.HasPrefix(inpt, "list connections"):
			niy()
		case strings.HasPrefix(inpt, "add subscription"):
			niy()
		case strings.HasPrefix(inpt, "add connection"):
			niy()
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
		default:
			fmt.Println("unknown command:", inpt)
			continue // no errors, no history
		}
		if err != nil {
			log.Print("error: ", err)
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
	"list subscriptions",
	"list connections",
	"list",
	"add subscription",
	"add connection",
	"remove subscription",
	"data size",
	"help",
	"exit",
	"quit",
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
	list connections [args todo]
		list connections of ...
	add subscription [args todo]
		add subscription ...blah
	add connection [args todo]
		add conection ...blah
	remove subscription
		remove subscription from ... blah
	data size
		get data size of ... blah
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
	Addr string
}

func listSubscriptions(client *Client, nodeId string) error {
	if nodeId == "" {
		return errors.New("node id required: list subscriptions <node id>")
	}
	// TODO: sanitize nodeId
	req := client.Addr + "/manager/nodes/" + nodeId + "/subscriptions"
	// TODO: log level
	log.Print("[GET] ", req)
	resp, err := client.Get(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// debug log
		log.Print("response status: ", resp.Status)
		// read error from body
		errb := make([]byte, ERROR_BODY_LEN)
		n, err := resp.Body.Read(errb)
		if err == nil {
			return errors.New("response error: " + string(errb[:n]))
		}
		if err != io.EOF {
			return errors.New("error reading response: " + err.Error())
		}
		// EOF: bad request or internal server error or somthing other error
		// that returns non-200 status and empty body
		return errors.New("error response status: " + resp.Status)
	}
	// temporary: todo: parse JSON
	_, err = io.Copy(os.Stdout, resp.Body)
	return err
}
