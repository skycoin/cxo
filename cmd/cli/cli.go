package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterh/liner"
)

const (
	DEFAULT_CXOD_ADDR = "http://127.0.0.1:6481"
	DEFAULT_TIMEOUT   = 0

	HISTORY_FILE   = ".cxo_cli_history"
	ERROR_BODY_LEN = 500
)

var ErrUnknownCommand = errors.New("unknown command")

func main() {
	// initialize logger
	log.SetFlags(log.LstdFlags)

	// exit code
	var code int = 0
	defer func() { os.Exit(code) }()

	// flags
	var (
		addr    string
		timeout time.Duration
		help    bool
		debug   bool
		execute string
	)
	// parse command line flags
	flag.StringVar(&addr,
		"a",
		DEFAULT_CXOD_ADDR,
		"server address")
	flag.DurationVar(&timeout,
		"t",
		DEFAULT_TIMEOUT,
		"request/response timeout")
	flag.BoolVar(&help,
		"h",
		false,
		"show help")
	flag.BoolVar(&debug,
		"d",
		false,
		"print debug logs")
	flag.StringVar(&execute,
		"e",
		"",
		"execute given command and exit")
	flag.Parse()

	// take a look at the flags
	if help {
		flag.PrintDefaults()
		return
	}

	var (
		client *Client
		err    error
	)

	// http client
	client = NewClient(addr, debug, timeout)

	if execute != "" {
		if _, err = client.executeCommand(execute, nil); err != nil {
			log.Print(err)
			code = 1
		}
		return
	}

	// liner
	var line = liner.NewLiner()
	defer line.Close()

	readHistory(line)
	defer storeHistory(line)

	line.SetCtrlCAborts(true)
	line.SetCompleter(autoComplite)
	line.SetTabCompletionStyle(liner.TabPrints)

	log.Print("starting client")
	log.Print("address:    ", addr)
	log.Print("timeout:    ", humanDuration(timeout, "no limits"))
	log.Print("debug logs: ", debug)

	fmt.Println("enter 'help' to get help")

	var inpt string
	var terminate bool

	// prompt loop
	for {
		inpt, err = line.Prompt("> ")
		if err != nil {
			log.Print("fatal: ", err)
			code = 1
			return
		}
		// TODO
		terminate, err = client.executeCommand(inpt, line)
		if err != nil && err != ErrUnknownCommand {
			log.Println(err)
		}
		if terminate {
			return
		}
	}
}

func (client *Client) executeCommand(inpt string,
	line *liner.State) (terminate bool, err error) {

	inpt = strings.TrimSpace(strings.ToLower(inpt))
	switch {

	case strings.HasPrefix(inpt, "list subscriptions"):
		err = client.listSubscriptions()

	case strings.HasPrefix(inpt, "list subscribers"):
		err = client.listSubscribers()

	case strings.HasPrefix(inpt, "list"):
		fmt.Println(`list what?
	- list subscriptions
	- list subscribers`)
		err = ErrUnknownCommand
		return

	case strings.HasPrefix(inpt, "add subscription"):
		err = client.addSubscription(trim(inpt, "add subscription"))

	case strings.HasPrefix(inpt, "add"):
		fmt.Println(`do you mean 'add subscription'?`)
		err = ErrUnknownCommand
		return

	case strings.HasPrefix(inpt, "remove subscription"):
		err = client.removeSubscription(trim(inpt, "remove subscription"))

	case strings.HasPrefix(inpt, "remove subscriber"):
		err = client.removeSubscriber(trim(inpt, "remove subscriber"))

	case strings.HasPrefix(inpt, "remove"):
		fmt.Println(`remove what?
	- remove subscription
	- remove subscriber`)
		err = ErrUnknownCommand
		return

	case strings.HasPrefix(inpt, "stat"):
		err = client.getStat()

	case strings.HasPrefix(inpt, "info"):
		err = client.getNodeInfo()

	case strings.HasPrefix(inpt, "close"):
		err = client.closeDaemon()

	case strings.HasPrefix(inpt, "help"):
		printHelp()

	case strings.HasPrefix(inpt, "exit"):
		fallthrough

	case strings.HasPrefix(inpt, "quit"):
		fmt.Println("cya")
		terminate = true
		return

	case inpt == "":
		return // do noting properly

	default:
		fmt.Println("unknown command:", inpt)
		err = ErrUnknownCommand
		return // no history

	}
	if line != nil {
		line.AppendHistory(inpt)
	}
	return
}

// utility, printf and break line
func printf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
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
	"list subscribers ",
	"list ",
	"add subscription ",
	"remove subscription ",
	"remove subscriber",
	"stat ",
	"info",
	"close",
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

func printHelp() {
	fmt.Print(`Available commands:

	list subscriptions
		list all subscriptions
	list subscribers
		list all subscribers
	add subscription <address> [desired public key]
		add subscription to given address, the public key is optional
	remove subscription <id or address>
		remove subscription by id or address
	remove subscriber <id or address>
		remove subscriber by id or address
	stat
		get statistic (total objects, memory) of all objects
	info
		print node id and address
	close
		terminate daemon
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
