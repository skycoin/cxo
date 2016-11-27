package schema

import (
	"fmt"
)

type synchronizer struct {
	store *Store //Target storage for synchronization
}

type SyncInfo struct {
	Total    int //Total objects in a system
	Shortage int //Objects need to be synchronized
}

func Synchronizer(store *Store) *synchronizer {
	return &synchronizer{store:store}
}

func (s *synchronizer) Sync(source *Store, ref Href) (<- chan SyncInfo, <- chan bool) {
	ch := make(chan SyncInfo)
	done := make(chan bool)
	go func() {
		defer close(done)
		defer close(ch)

		destInfo := HrefInfo{}
		ref.Expand(s.store, &destInfo)

		for (len(destInfo.No )> 0){
			for i := 0; i < len(destInfo.No); i++ {
				synchData(s.store, source, destInfo.No[i])
			}
			destInfo = HrefInfo{}
			ref.Expand(s.store, &destInfo)
		}
		done <- true
	}()
	return ch, done
}
//

func synchData(d, s *Store, key HKey) bool {
	if (!d.Has(key)) {
		obj, ok := s.Get(key)
		if (!ok){
			fmt.Println("Object not found")
		}
		d.Add(key, obj)
		return true
	}
	return false
}
