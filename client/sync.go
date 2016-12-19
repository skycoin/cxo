package client

import (
	"github.com/skycoin/cxo/gui"
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
	"fmt"
)

type syncContext struct {
	messenger *gui.Messenger
	pull      map[cipher.SHA256]chan bool
	mu        *sync.RWMutex
}

func SyncContext(messenger *gui.Messenger) *syncContext {
	return &syncContext{messenger:messenger, mu:&sync.RWMutex{}}
}

func (s *syncContext) Accept(hash cipher.SHA256, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.pull[hash]
	if (ok) {
		s.pull[hash] <- true
		close(s.pull[hash])
		delete(s.pull, hash);
	}
}
func (s syncContext) Request(hash cipher.SHA256) <- chan bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.pull[hash]

	//Hash is a lready in query
	if (ok) {
		return s.pull[hash]
	}

	s.pull[hash] = make(chan bool)

	go func() {
		err := s.messenger.Request(hash)
		if (err != nil) {
			fmt.Errorf("Error requestin data: %v", hash)
			s.pull[hash] <- false
			close(s.pull[hash])
			delete(s.pull, hash);

		}
	}()
	return s.pull[hash]
}
