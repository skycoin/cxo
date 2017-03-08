package main

import "fmt"

func hello(client *Client, args []string) {
	res, e := client.SendToRPC("Greet", args)
	if e != nil {
		fmt.Println(e)
	}

	var resMsg string
	res.Deserialize(&resMsg)
	fmt.Println(resMsg)
}

func addBoard(c *Client, args []string) {
	_, e := c.SendToRPC("AddBoard", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println("Board successfully added.")
}

func removeBoard(c *Client, args []string) {
	_, e := c.SendToRPC("RemoveBoard", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println("Board successfully removed.")
}

func listBoards(c *Client, args []string) {
	msg, e := c.SendToRPC("ListBoards", args)
	if e != nil {
		fmt.Println(e)
		return
	}
	fmt.Println(msg)
}

func addThread(c *Client, args []string) {
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

func removeThread(c *Client, args []string) {
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
