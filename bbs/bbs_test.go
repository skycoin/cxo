package bbs

import (
	"testing"
	"github.com/skycoin/cxo/data"
	"fmt"
	"time"
	"math/rand"
	"github.com/skycoin/skycoin/src/cipher"
)

func Test_Bbs_1(T *testing.T) {
	db := data.NewDB()
	bbs := CreateBbs(db, &testNode{})
	post := bbs.CreatePost("Header post", "Header text")
	post2 := bbs.CreatePost("Header post2", "Header text2")
	th1 := bbs.CreateThread("Thread", post, post2)
	bbs.Container.Inspect()

	bbs.AddBoard("Test Board", th1)
	st := db.Statistic()
	fmt.Println(st)
	if (st.Total != 15) {
		T.Fatal("Invalid number of objects", st.Total)
	}
}
//
//func Test_Bbs_2(T *testing.T) {
//	db := data.NewDB()
//	bbs := prepareTestData(db)
//	fmt.Println(db.Statistic())
//	schema, _ := bbs.Container.GetSchemaKey("thread")
//	fmt.Println("schema", schema)
//	sm := bbs.Container.GetAllBySchema(schema)
//	if (len(sm) != 10) {
//		T.Fatal("Invalid number of threads", len(sm))
//	}
//
//	//res:= []interface{}{}
//	for _, k := range sm {
//		ref := skyobject.HashLink{Ref:k}
//		ref.SetData(k[:])
//		fmt.Println(ref.String(bbs.Container))
//		//fields := bbs.Container.LoadFields(k)
//		//res = append(res, fields)
//		//fmt.Println(fields)
//	}
//}

func prepareTestData(ds data.IDataSource) *Bbs {
	bbs := CreateBbs(ds, &testNode{})
	boards := []Board{}
	for b := 0; b < 1; b++ {
		threads := []Thread{}
		for t := 0; t < 10; t++ {
			posts := []Post{}
			for p := 0; p < 10; p++ {
				posts = append(posts, bbs.CreatePost("Post_" + GenerateString(15), "some Text"))
			}
			threads = append(threads, bbs.CreateThread("Thread_" + GenerateString(15), posts...))
		}
		boards = append(boards, bbs.AddBoard("Board_" + GenerateString(15), threads...))
	}
	return bbs
}

type testNode struct {

}

func (t *testNode) Sign(hash cipher.SHA256) cipher.Sig {
	_, sk := cipher.GenerateKeyPair()
	return cipher.SignHash(hash, sk)
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
