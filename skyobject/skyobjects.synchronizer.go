package skyobject

import (
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
)

type ISynchronizer interface {
	Sync(root HashRoot) (<- chan bool)
}

type ISyncRemote interface {
	Request(hash cipher.SHA256) <- chan bool
}

type synchronizer struct {
	container ISkyObjects
	remote    ISyncRemote
}

func Synchronizer(container ISkyObjects, remote ISyncRemote) *synchronizer {
	return &synchronizer{container:container, remote:remote}
}

func (s *synchronizer) Sync(root HashRoot) (<- chan bool) {
	done := make(chan bool)
	go func() {
		defer close(done)
		s.container.Inspect()
		fmt.Println("CheckKey", root.Ref)
		res := s.CheckKey(s.remote, root.Ref)
		fmt.Println("Sync Result: ", res)
		done <- true
	}()
	return done
}

func (s *synchronizer) CheckKey(remote ISyncRemote, key cipher.SHA256) bool {
	if !s.container.Has(key) {
		fmt.Println("Requesting key: ", key)
		ok := <-remote.Request(key)
		if (!ok) {
			return false
		}
		fmt.Println("Recieved   key: ", key)
	}

	r := Href{Ref:key}
	for k := range r.References(s.container) {
		fmt.Println("Inner Reference: ", k)
		s.CheckKey(remote, k)
	}

	return true
}
