package bbs

import (
	"testing"
	"github.com/skycoin/cxo/schema"
	"time"
	"math/rand"
	"fmt"
	"github.com/skycoin/cxo/data"
)

type TestObject1 struct {
	Field1 int32
	Field2 []byte
}

type TestObject2 struct {
	Field1 uint32
	//Field2 schema.IHashList
	Field3 uint32
}

type TestRoot struct {
	Field1 TestObject2
}

func Test_Bbs_1(T *testing.T) {
	bbs := CreateBbs(data.NewDB())
	board, _ := bbs.AddBoard("Test Board")
	bbs.AddBoard("Another Board")

	var b Board
	schema.Href{Hash:board}.ToObject(bbs.Container, &b)

	bbs.AddThread(board, "Thread 1")

	board = bbs.GetBoard("Test Board")
	bbs.AddThread(board, "Thread 2")

	board = bbs.GetBoard("Test Board")
	bbs.AddThread(board, "Thread 3")

	boards := bbs.AllBoars()
	if (boards[0].Name != "Test Board") {
		T.Fatal("Board name is not equal")
	}
	//
	threads := bbs.GetThreads(boards[0])
	if (threads[0].Name != "Thread 1") {
		T.Fatal("Thread name is not equal")
	}

	if (threads[2].Name != "Thread 3") {
		T.Fatal("Thread name is not equal")
	}
}

func Test_Bbs_Syncronization(T *testing.T) {
	//1: Create Bbs1 and Fill the date
	bbs1 := prepareTestData()
	info := schema.HrefInfo{}

	bbs1.Container.Root.Expand(bbs1.Container, &info)
	fmt.Println(len(info.Has))
	if (len(info.Has) != 368) {
		T.Fatal("Number of objects in system is not equal")
	}

	//2: Crate empty Bbs2
	bbs2 := CreateBbs(data.NewDB())
	//3: Synchronize data
	syncronizer := schema.Synchronizer(bbs2.Container)

	bbs2.Container.Root = bbs1.Container.Root
	progress, done := syncronizer.Sync(bbs1.Container, bbs1.Container.Root)

	for {
		select {
		case sinfo := <-progress:
			if (sinfo.Total != 0) {
				fmt.Println("Total: ", sinfo.Total, ", Done: ", 100 * (sinfo.Total - sinfo.Shortage) / sinfo.Total)
			} else {
				fmt.Println("Total: ", sinfo.Total)
			}
		case <-done:
			info = schema.HrefInfo{}
			bbs2.Container.Root.Expand(bbs2.Container, &info)
			fmt.Println("Have:", len(info.Has), "haven't: ", len(info.No))
			fmt.Println("Done!")

			if (len(info.Has) != 368) {
				T.Fatal("Number of objects in system is not equal")
			}

			if (len(info.No) != 0) {
				T.Fatal("Invalid number of missing objects")
			}
			return
		}
	}
	//4: Validate
	//5: Add data to the Bbs2
	//6: Synchronize data
	//7: Validate
}

func prepareTestData() *Bbs {
	bbs := CreateBbs(data.NewDB())
	boards := []Board{}
	for b := 0; b < 3; b++ {
		threads := []Thread{}
		for t := 0; t < 10; t++ {
			posts := []Post{}
			for p := 0; p < 10; p++ {
				posts = append(posts, Post{Text: "Post_" + GenerateString(15)})
			}
			threads = append(threads, bbs.CreateThread("Thread_" + GenerateString(15), posts...))
		}
		boards = append(boards, bbs.CreateBoard("Board_" + GenerateString(15), threads...))
	}
	bbs.AddBoards(boards)
	return bbs
}

const letterBytes = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1 << letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func GenerateString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n - 1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
