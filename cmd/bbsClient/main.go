package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	_Board  = "board"
	_Thread = "thread"
)

var (
	port = flag.String("port", "1235", "port to connect to BBS server")
)

func main() {
	rpcClient := RunClient(":" + *port)
	fmt.Println("\nEnter your command (type help for commands):")
	for {
		if commandDispatcher(rpcClient) {
			break
		}
	}
}

func commandDispatcher(c *Client) bool {
	cmd, args := cliInput("\n> ")
	fmt.Print("\n")
	switch strings.ToLower(cmd) {
	case "":
		break
	case "exit", "quit":
		fmt.Print("Goodbye!\n\n")
		return true
	case "help":
		printHelp()
	case "hello":
		hello(c, args)
	case "add":
		caseAdd(c, args)
	case "remove":
		caseRemove(c, args)
	case "list":
		caseList(c, args)
	default:
		fmt.Printf("Unknown command: %s, type 'help' to get the list of available commands.\n\n", cmd)
	}
	return false
}

func caseAdd(c *Client, args []string) {
	cmd, args := args[0], args[1:]
	switch cmd {
	case _Board:
		addBoard(c, args)
	case _Thread:
		addThread(c, args)
	default:
		fmt.Printf("Unknown command 'add %s'.\n\n", cmd)
		fmt.Println("Avaliable:")
		fmt.Println(" - add board <name> <description>")
		fmt.Print("\n")
	}
}

func caseRemove(c *Client, args []string) {
	cmd, args := args[0], args[1:]
	switch cmd {
	case _Board:
		removeBoard(c, args)
	case _Thread:
		removeThread(c, args)
	default:
		fmt.Printf("Unknown command 'remove %s'.\n\n", cmd)
		fmt.Println("Avaliable:")
		fmt.Println(" - remove board <name>")
		fmt.Print("\n")
	}
}

func caseList(c *Client, args []string) {
	cmd, args := args[0], args[1:]
	switch cmd {
	case "boards", _Board:
		listBoards(c, args)
	default:
		fmt.Println("Screwed up.")
	}
}

func cliInput(prompt string) (command string, args []string) {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	var moveOn = true
	var bCount = 0
	for i, c := range input {
		if moveOn {
			args = append(args, "")
			moveOn = false
		}
		switch c {
		case rune('"'):
			bCount++
		case rune(' '):
			if input[i-1] == ' ' {
				continue
			}
			moveOn = (bCount%2 == 0) // only move on if not in params.
			if moveOn == false {
				args[len(args)-1] += string(c)
			}
		default:
			args[len(args)-1] += string(c)
		}
	}
	command = args[0]
	args = args[1:]
	return
}

func printHelp() {

	fmt.Print("\n=====================\n")
	fmt.Print("HELP\n")
	fmt.Print("=====================\n\n")

	fmt.Print("help\t\tprints help.\n\n")

	fmt.Print("hello X\t\tsends a hello to X.\n")
	fmt.Print("random\t\tadds random data (boards, threads and posts).\n\n")

	fmt.Print("list boards\t\tlists all boards as keys.\n")
	fmt.Print("list threads\t\tlists all threads as keys.\n")
	fmt.Print("list threads X\t\tlists all threads of board X as keys.\n")
	fmt.Print("list posts\t\tlists all posts as IDs.\n")
	fmt.Print("list posts X\t\tlists all posts of thread X as keys.\n\n")

	fmt.Print("exit (or quit)\t\tcloses the terminal.\n\n")
}
