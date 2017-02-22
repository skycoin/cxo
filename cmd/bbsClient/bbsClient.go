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

type objectLink struct {
	ID   string
	Name string
}

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

	case "random":
		generateRandomData(rpcClient)

	case "list":
		switch {
		case len(args) < 1:
			fmt.Printf("\nUnspecified arguments for 'list'.\n\n")
			break

		case args[0] == "boards":
			listBoards(rpcClient)

		case args[0] == "threads":
			listThreads(rpcClient, args[1:])

		case args[0] == "posts":
			listPosts(rpcClient, args[1:])

		default:
			fmt.Printf("\nUnknown arguments for 'list': %v, type 'help' to get the list of available commands.\n\n", args)
		}

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

	fmt.Println("help\t\tprints help.\n")

	fmt.Println("hello X\t\tsends a hello to X.")
	fmt.Println("random\t\tadds random data (boards, threads and posts).\n")

	fmt.Println("list boards\t\tlists all boards as keys.")
	fmt.Println("list threads\t\tlists all threads as keys.")
	fmt.Println("list threads X\t\tlists all threads of board X as keys.")
	fmt.Println("list posts\t\tlists all posts as IDs.")
	fmt.Println("list posts X\t\tlists all posts of thread X as keys.\n")

	fmt.Println("exit (or quit)\t\tcloses the terminal.\n")
}

func hello(client *RPCClient, args []string) {
	response, e := client.SendToRPC("Greet", args)
	if e != nil {
		fmt.Println("ERROR:", e)
	}

	var respMsg string
	e = messages.Deserialize(response, &respMsg)
	if e != nil {
		fmt.Println("ERROR:", e)
	}

	fmt.Println(respMsg)

}

func listBoards(client *RPCClient) {
	response, e := client.SendToRPC("ListBoards", []string{})
	if e != nil {
		fmt.Println("ERROR:", e)
		return
	}

	var respArray []objectLink
	e = messages.Deserialize(response, &respArray)
	if e != nil {
		fmt.Println("ERROR:", e)
		return
	}

	switch {
	case len(respArray) < 1:
		fmt.Println("No boards to display.")

	default:
		fmt.Println("Listing", len(respArray), "boards:")
		for _, v := range respArray {
			fmt.Println("", "-", v.Name)
		}
	}
}

func listThreads(client *RPCClient, args []string) {
	response, e := client.SendToRPC("ListThreads", args)
	if e != nil {
		fmt.Println("ERROR:", e)
		return
	}

	var respArray []objectLink
	e = messages.Deserialize(response, &respArray)
	if e != nil {
		fmt.Println("ERROR:", e)
		return
	}

	switch {
	case len(respArray) < 1:
		fmt.Println("No threads to display.")

	default:
		fmt.Println("Listing", len(respArray), "threads:")
		for _, v := range respArray {
			fmt.Println("", "-", v.Name)
		}
	}
}

func listPosts(client *RPCClient, args []string) {
	response, e := client.SendToRPC("ListPosts", args)
	if e != nil {
		fmt.Println("ERROR:", e)
		return
	}

	var respArray []objectLink
	e = messages.Deserialize(response, &respArray)
	if e != nil {
		fmt.Println("ERROR:", e)
		return
	}

	switch {
	case len(respArray) < 1:
		fmt.Println("No posts to display.")

	default:
		fmt.Println("Listing", len(respArray), "posts:")
		for _, v := range respArray {
			fmt.Println("", "-", v.Name)
		}
	}
}

func generateRandomData(client *RPCClient) {
	response, e := client.SendToRPC("GenerateRandomData", []string{})
	if e != nil {
		fmt.Println("ERROR:", e)
		return
	}

	var respMsg string
	e = messages.Deserialize(response, &respMsg)
	if e != nil {
		fmt.Println("ERROR:", e)
	}

	fmt.Println(respMsg)
}
