package main

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// AddThread adds a thread to specified board.
// arg0 : Thread Name (new).
// arg1 : Board Name (existing).
func (r *RPCReceiver) AddThread(args []string, result *[]byte) error {
	if len(args) != 2 {
		return fmt.Errorf("invalid number of arguments")
	}
	if e := r.I.AddThreadToBoard(args[0], args[1]); e != nil {
		return e
	}
	fmt.Println("[AddThread] Added:", args[0], "to", args[1])
	return nil
}

// RemoveThread removes a thread from a specified board.
// arg0 : Thread Name (existing).
// arg1 : Board Name (existing).
func (r *RPCReceiver) RemoveThread(args []string, result *[]byte) error {
	if len(args) != 2 {
		return fmt.Errorf("invalid number of arguments")
	}
	if e := r.I.RemoveThreadFromBoard(args[0], args[1]); e != nil {
		return e
	}
	fmt.Println("[RemoveThread] Removed:", args[0], "to", args[1])
	return nil
}

// ListThreads lists all the keys of threads from specified board.
// arg0: Board Name (existing).
func (r *RPCReceiver) ListThreads(args []string, result *[]byte) error {
	if len(args) != 1 {
		return fmt.Errorf("invalid number of arguments")
	}
	boardRef, _ := r.I.FindBoard(args[0])
	if boardRef == nil {
		return fmt.Errorf("board of name '%s' not found", args[0])
	}
	threadKeys := r.I.GetThreadIndexesFromBoard(boardRef.GetKey())
	*result = encoder.Serialize(&threadKeys)
	return nil
}

// DownloadThreads downloads all threads of given keys.
func (r *RPCReceiver) DownloadThreads(args []string, result *[]byte) error {
	// Get thread keys.
	var threadKeys []cipher.SHA256
	for _, k := range args {
		key, e := cipher.SHA256FromHex(k)
		if e != nil {
			fmt.Println("[DownloadThreads] Error:", e)
		}
		threadKeys = append(threadKeys, key)
	}
	// Get threads.
	threads := r.I.GetThreads(threadKeys)
	*result = encoder.Serialize(&threads)
	defer fmt.Println("[DownloadThreads] Downloaded:", len(threads))
	return nil
}
