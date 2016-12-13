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


type ISyncTarget interface {
	Has(key cipher.SHA256) bool
	Add(key cipher.SHA256, data []byte)
}

type ISyncSource interface {
	Get(key cipher.SHA256) []byte
	Referencies(key cipher.SHA256) []cipher.SHA256
}

type synchronizer struct {
	container ISkyObjects //Target storage for synchronization
	target    ISyncTarget
}

type SyncInfo struct {
	Total    int //Total objects in a system
	Shortage int //Objects need to be synchronized
}
//
func Synchronizer(container ISkyObjects) *synchronizer {
	return &synchronizer{container:container}
}

func SynchronizerR(container ISkyObjects, target ISyncTarget) *synchronizer {
	return &synchronizer{target:target, container:container}
}

func (s *synchronizer) SyncRoot(source ISyncSource, root HashRoot) (<- chan SyncInfo, <- chan bool) {
	ch := make(chan SyncInfo)
	done := make(chan bool)
	go func() {
		defer close(done)
		defer close(ch)
		//
		missing := getMissingKeysR(s.target, source, Href(root))
		for (len(missing) > 0) {
			fmt.Println(missing)
			for _, key := range missing {
				s.target.Add(key, source.Get(key))
			}
			missing = getMissingKeysR(s.target, source, Href(root))
		}
		fmt.Println("Done")
		done <- true
	}()
	return ch, done
}

func getMissingKeysR(target ISyncTarget, source ISyncSource, ref Href) []cipher.SHA256 {
	keys := source.Referencies(ref.Ref)
	result := []cipher.SHA256{}
	for _, key := range keys {
		if (!target.Has(key)) {
			result = append(result, key)
		}
	}
	return result
}

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

func getMissingKeys(target ISkyObjects, ref Href) []cipher.SHA256 {
	keys := ref.References(target)
	result := []cipher.SHA256{}
	for key := range keys {
		if (!target.Has(key)) {
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
