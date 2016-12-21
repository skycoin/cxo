package skyobject

import (
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
	"time"
)

type ISynchronizer interface {
	Sync(hash cipher.SHA256, requester func(cipher.SHA256) error) bool
}

type ISyncRemote interface {
	Request(hash cipher.SHA256) bool
}

type synchronizer struct {
	container ISkyObjects
}

func Synchronizer(container ISkyObjects) *synchronizer {
	return &synchronizer{container:container}
}
//
//func (s *synchronizer) Sync2(hash cipher.SHA256) (<- chan bool) {
//	fmt.Println(s.container.Statistic())
//	//s.container.Inspect()
//	done := make(chan bool)
//	go func() {
//		defer close(done)
//		res := s.CheckKey(s.remote, hash, 0)
//		fmt.Println("Sync Result: ", res)
//		s.container.Statistic()
//		done <- true
//	}()
//	return done
//}

func (s *synchronizer) Sync(hash cipher.SHA256, requester func(cipher.SHA256) error) bool {
	fmt.Println(s.container.Statistic())
	//s.container.Inspect()
	//done := make(chan bool)
	//go func() {
	//	defer close(done)
	return s.CheckKey(requester, hash, 0)
	//fmt.Println("Sync Result: ", res)
	//s.container.Statistic()
	//done <- true
	//}()
	//return done
}

func (s *synchronizer) CheckKey(requester func(cipher.SHA256) error, key cipher.SHA256, step int) bool {
	if ( step == 20) {
		return false
	}

	data, ok := s.container.Get(key)
	if (!ok) {
		fmt.Println("Requesting key: ", key)
		error := requester(key)
		time.Sleep(time.Millisecond * 500)
		if (error != nil) {
			return false
		}
		fmt.Println("Recieved   key: ", key)
		return true
	}

	typeKey := cipher.SHA256{}
	typeKey.Set(data[:32])

	if (typeKey != _schemaType) {
		s.CheckKey(requester, typeKey, step + 1)
		time.Sleep(time.Millisecond * 500)
		r := Href{Ref:key}
		for k := range r.References(s.container) {
			if (k != key) {
				s.CheckKey(requester, k, step + 1)
				time.Sleep(time.Millisecond * 500)
			}
		}
	}
	return true
}
