package main

import (
	"fmt"

	"github.com/skycoin/cxo/bbs"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

func hello(client *RPCClient, args []string) {
	response, e := client.SendToRPC("Greet", args)
	if e != nil {
		fmt.Println("ERROR:", e)
	}

	var respMsg string
	e = encoder.DeserializeRaw(response, &respMsg)
	if e != nil {
		fmt.Println("ERROR:", e)
	}

	fmt.Println(respMsg)

}

func addBoard(c *RPCClient, args []string) {
	_, e := c.SendToRPC("AddBoard", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println("Board successfully added.")
}

func removeBoard(c *RPCClient, args []string) {
	_, e := c.SendToRPC("RemoveBoard", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println("Board successfully removed.")
}

func listBoards(c *RPCClient, args []string) {
	msg, e := c.SendToRPC("ListBoards", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	var boardKeys []cipher.SHA256
	encoder.DeserializeRaw(msg, &boardKeys)
	tree.AddBranches(boardKeys)

	// Download boards.
	args = []string{}
	dlBoardKeys := tree.GetUndownloadedBoardKeys()
	for _, key := range dlBoardKeys {
		args = append(args, key.Hex())
	}
	msg, e = c.SendToRPC("DownloadBoards", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	var dlBoards []bbs.SBoard
	encoder.DeserializeRaw(msg, &dlBoards)
	tree.AddDownloadedBoards(dlBoardKeys, dlBoards)

	// Print results.
	fmt.Println("Boards:")
	for key := range tree.Branches {
		board := tree.GetBoard(key)
		fmt.Printf(" [%s] '%s'\n", board.Name, board.Description)
	}
}

func addThread(c *RPCClient, args []string) {
	if len(args) != 3 || args[1] != "to" {
		fmt.Println("Format needs to be: 'add thread <new_thread_name> to <existing_board_name>'.")
		return
	}
	args = []string{args[0], args[2]}
	_, e := c.SendToRPC("AddThread", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println("Thread successfully added to Board.")
}

func removeThread(c *RPCClient, args []string) {
	if len(args) != 3 || args[1] != "from" {
		fmt.Println("Format needs to be: 'remove thread <existing_thread_name> from <existing_board_name>'.")
		return
	}
	args = []string{args[0], args[2]}
	_, e := c.SendToRPC("RemoveThread", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println("Thread successfully removed from Board.")
}
