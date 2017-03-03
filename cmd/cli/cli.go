package main

import (
	"errors"
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"strings"

	"github.com/peterh/liner"
)

//
// "rpc.Connect",    "127.0.0.1:9090", *error
// "rpc.Disconnect", "127.0.0.1:9090", *error
// "rpc.List",       struct{}{},       *struct{List []string, Err error}
// "rpc.Info",       struct{}{}.       *struct{
//                                         Address string,
//                                         Stat    struct{
//                                             Total  int
//                                             Memory int
//                                         },
//                                         Err error,
//                                      }
//

var (
	ErrUnknowCommand    = errors.New("unknown command")
	ErrMisisngArgument  = errors.New("missing argument")
	ErrTooManyArguments = errors.New("too many arguments")

	commands = []string{
		"list",
		"connect",
		"disconnect",
		"info",
		"quit",
		"exit",
	}
)

func main() {
	var (
		address string
		execute string

		client *rpc.Client
		err    error

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
		"execute commant and exit")

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

	if client, err = rpc.Dial("tcp", address); err != nil {
		fmt.Fprintln(os.Stderr, "error creating rpc-clinet:", err)
		code = 1
		return
	}
	defer client.Close()

	if execute != "" {
		_, err = executeCommand(execute, client, nil)
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

	// rpompt loop

	fmt.Println("enter 'help' to get help")
	for {
		cmd, err = line.Prompt("> ")
		if err != nil && err != liner.ErrPromptAborted {
			fmt.Fprintln(os.Stderr, "fatal error:", err)
			code = 1
			return
		}
		terminate, err = executeCommand(cmd, client, line)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		if terminate {
			return
		}
		line.AppendHistory(cmd)
	}

}

func args(ss []string) (string, error) {
	switch len(ss) {
	case 0, 1:
		return "", ErrMisisngArgument
	case 2:
		return ss[1], nil
	default:
	}
	return "", ErrTooManyArguments
}

func executeCommand(command string, client *rpc.Client,
	line *liner.State) (terminate bool, err error) {

	ss := strings.Fields(command)
	if len(ss) == 0 {
		return
	}
	switch ss[0] {
	case "list":
		err = list(client)
	case "connect":
		err = connect(client, ss)
	case "disconnect":
		err = disconnect(client, ss)
	case "info":
		err = info(client)
	case "help":
		showHelp()
	case "quit":
		fallthrough
	case "exit":
		terminate = true
		fmt.Println("cya")
		return
	default:
		err = ErrUnknowCommand
		return
	}
	return
}

func showHelp() {
	fmt.Println(`

	list
		list connections
	connect <address>
		add connection to given address
	disconnect	<address>
		disconnect from given address
	info
		obtain information and statistic about node
	help
		show this help message
	quit or exit
		leave the cli

`)
}

func list(client *rpc.Client) (err error) {
	var reply = new(ListReply)
	if err = client.Call("rpc.List", struct{}{}, reply); err != nil {
		return
	}
	if len(reply.List) == 0 {
		fmt.Println("  there aren't connections")
		return
	}
	for _, c := range reply.List {
		fmt.Println("  -", c)
	}
	return
}

func info(client *rpc.Client) (err error) {
	var reply = new(InfoReply)
	if err = client.Call("rpc.Info", struct{}{}, reply); err != nil {
		return
	}
	fmt.Printf(`  Address:       %s
  Total objects: %d
  Memory:        %d
`,
		reply.Address,
		reply.Stat.Total,
		reply.Stat.Memory)
	return

}

func connect(client *rpc.Client, ss []string) (err error) {
	var address string
	if address, err = args(ss); err != nil {
		return
	}
	if err = client.Call("rpc.Connect", address, &struct{}{}); err != nil {
		return
	}
	fmt.Println("connected")
	return
}

func disconnect(client *rpc.Client, ss []string) (err error) {
	var address string
	if address, err = args(ss); err != nil {
		return
	}
	if err = client.Call("rpc.Disconnect", address, &struct{}{}); err != nil {
		return
	}
	fmt.Println("disconnected")
	return
}

//
// types
//

type ListReply struct {
	List []string
	Err  error
}

type InfoReply struct {
	Address string
	Stat    struct {
		Total  int
		Memory int
	}
	Err error
}
