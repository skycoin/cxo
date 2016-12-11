package skyobject

import (
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
)

//import (
//	"fmt"
//	"github.com/skycoin/skycoin/src/cipher"
//)
//
type synchronizer struct {
	container ISkyObjects //Target storage for synchronization
}

type SyncInfo struct {
	Total    int //Total objects in a system
	Shortage int //Objects need to be synchronized
}
//
func Synchronizer(container ISkyObjects) *synchronizer {
	return &synchronizer{container:container}
}
//
func (s *synchronizer) Sync(source ISkyObjects, ref Href) (<- chan SyncInfo, <- chan bool) {
	ch := make(chan SyncInfo)
	done := make(chan bool)
	go func() {
		defer close(done)
		defer close(ch)

		missing := getMissingKeys(source, ref)
		//destInfo := HrefInfo{}
		//ref.Expand(s.store, &destInfo)
		//
		for (len(missing) > 0) {
			fmt.Println(missing)

			for _, key := range missing {
				synchData(s.container, source, key)
			}
			missing = getMissingKeys(source, ref)
		}
		fmt.Println("Done")
		done <- true
	}()
	return ch, done
}

func getMissingKeys(source ISkyObjects, ref Href) []cipher.SHA256 {
	keys := ref.References(source)
	result := []cipher.SHA256{}
	for _, key := range keys {
		if (!source.Has(key)) {
			result = append(result, key)
		}
	}
	return result
}

//
func synchData(d, s ISkyObjects, key cipher.SHA256) bool {
	if (!d.Has(key)) {
		obj, ok := s.Get(key)
		if (!ok) {
			fmt.Println("Object not found")
		}
		d.SaveData(obj)
		return true
	}
	return false
}
