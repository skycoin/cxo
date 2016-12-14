package client

import (
	"github.com/skycoin/cxo/gui"
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
)

type syncContext struct {
	messenger *gui.Messenger
	pull      map[cipher.SHA256]chan bool
	mu        *sync.RWMutex
}

func SyncContext(messenger *gui.Messenger) *syncContext {
	return &syncContext{messenger:messenger}
}

func (s *syncContext) Sync(hash cipher.SHA256) <- chan bool {
	s.mu.Lock()
	_, ok := s.pull[hash]
	if (!ok){
		s.pull[hash] = make(chan bool)
	}

	defer s.mu.Unlock()
	err := s.messenger.Announce(hash)
	if (err != nil) {
		s.pull[hash] <- true
	}
	return s.pull[hash]
}

//
//func (s *syncContext) Has(key cipher.SHA256) bool {
//	return false
//}
//
//func (s *syncContext) Add(key cipher.SHA256, data []byte) {
//
//}
//
//
//func (s *syncContext) Get(key cipher.SHA256) []byte{
//return nil
//}
//func (s *syncContext) Referencies(key cipher.SHA256) []cipher.SHA256{
//
//}
