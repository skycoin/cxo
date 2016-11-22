package bbs

import (
	"testing"
	"fmt"
)

func Test_Bbs_1(T *testing.T) {
	bbs := CreateBbs()
	board, _ := bbs.AddBoard("Test Board")
	bbs.AddBoard("Another Board")

	bbs.AddThread(board, "Thread 1")

	board = bbs.GetBoard("Test Board")
	bbs.AddThread(board, "Thread 2")


	board = bbs.GetBoard("Test Board")
	bbs.AddThread(board, "Thread 3")

	boards := bbs.AllBoars()
	fmt.Println("bbs.AllBoars()",boards)
	if (boards[0].Name != "Test Board") {
		T.Fatal("Board name is not equal")
	}

	threads := bbs.GetThreads(boards[0])
	fmt.Println("All threads", threads)
	if (threads[0].Name != "Thread 1"){
		T.Fatal("Thread name is not equal")
	}

	if (threads[2].Name != "Thread 3"){
		T.Fatal("Thread name is not equal")
	}
}
