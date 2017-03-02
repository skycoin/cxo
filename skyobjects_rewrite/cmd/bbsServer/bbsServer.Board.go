package main

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// AddBoard adds a board to root.
// arg1 : Board Name.
// arg2 : Board Description.
func (r *RPCReceiver) AddBoard(args []string, result *[]byte) error {
	if len(args) != 2 {
		return fmt.Errorf("invalid number of arguments")
	}
	boardKey, e := r.I.AddBoard(args[0], args[1])
	if e != nil {
		return e
	}
	fmt.Println("[AddBoard] Added:", args)
	*result = encoder.Serialize(&boardKey)
	return nil
}

// RemoveBoard removes a board from root.
// arg: Board Name.
func (r *RPCReceiver) RemoveBoard(args []string, result *[]byte) error {
	if len(args) != 1 {
		return fmt.Errorf("invalid number of arguments")
	}
	e := r.I.RemoveBoard(args[0])
	if e != nil {
		return e
	}
	fmt.Println("[RemoveBoard] Removed:", args)
	return nil
}

// ListBoards lists all the keys of the boards.
func (r *RPCReceiver) ListBoards(args []string, result *[]byte) error {
	if len(args) != 0 {
		return fmt.Errorf("invalid number of arguments")
	}
	keys := r.I.GetBoardIndexes()
	*result = encoder.Serialize(&keys)
	return nil
}

// DownloadBoards downloads all boards of given keys.
func (r *RPCReceiver) DownloadBoards(args []string, result *[]byte) error {
	// Get board keys.
	var boardKeys []cipher.SHA256
	for _, k := range args {
		key, e := cipher.SHA256FromHex(k)
		if e != nil {
			fmt.Println("[DownloadBoards] Error:", e)
		}
		boardKeys = append(boardKeys, key)
	}
	// Get boards.
	boards := r.I.GetBoards(boardKeys)
	*result = encoder.Serialize(&boards)
	defer fmt.Println("[DownloadBoards] Downloaded:", len(boards))
	return nil
}
