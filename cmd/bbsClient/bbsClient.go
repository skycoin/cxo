package main

import (
	"bufio"
	"fmt"
	// "github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/mesh/messages"
	// "net/rpc"
	"os"
	"strings"
)

func main() {
	port := "1235"
	if len(os.Args) >= 2 {
		port = os.Args[1]
	}
	rpcClient := RunClient(":" + port)
	promptCycle(rpcClient)
}

func promptCycle(rpcClient *RPCClient) {
	for {
		if commandDispatcher(rpcClient) {
			break
		}
	}
}

func commandDispatcher(rpcClient *RPCClient) bool {
	cmd, args := cliInput("\nEnter the command:\n>>> ")

	if cmd == "" {
		return false
	}

	cmd = strings.ToLower(cmd)

	switch cmd {
	case "exit", "quit":
		fmt.Println("\nGoodbye!\n")
		return true

	case "help":
		printHelp()

	case "hello":
		hello(rpcClient, args)

	default:
		fmt.Printf("\nUnknown command: %s, type 'help' to get the list of available commands.\n\n", cmd)

	}
	return false
}

func cliInput(prompt string) (command string, args []string) {
	fmt.Print(prompt)
	command = ""
	args = []string{}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()
	splitted := strings.Fields(input)
	if len(splitted) == 0 {
		return
	}
	command = splitted[0]
	if len(splitted) > 1 {
		args = splitted[1:]
	}
	return
}

func printHelp() {

	fmt.Println("\n=====================")
	fmt.Println("HELP")
	fmt.Println("=====================\n")

	fmt.Println("help\t\tprints help.")
	fmt.Println("hello X\t\tsends a hello.")
	fmt.Println("exit (or quit)\t\tcloses the terminal.\n")
}

func hello(client *RPCClient, args []string) {
	response, e := client.SendToRPC("Greet", args)
	if e != nil {
		fmt.Errorf("hello: %v", e)
	}

	var respMsg string
	e = messages.Deserialize(response, &respMsg)
	if e != nil {
		fmt.Errorf("hello: %v", e)
	}

	fmt.Println(respMsg)

}
