package schema

import (
	"fmt"
	"github.com/skycoin/skycoin/src/cipher"
	"sync"
	"github.com/skycoin/cxo/encoder"
	"errors"
)

type Store struct {
	data            map[cipher.SHA256][]byte
	mu              *sync.RWMutex
	newDataCallback func(cipher.SHA256, interface{}) error
}

func NewStore() *Store {
	db := Store{}
	db.data = make(map[cipher.SHA256][]byte)
	db.mu = &sync.RWMutex{}
	return &db
}

func (db *Store) Save(value interface{}) (cipher.SHA256, error) {
	data := encoder.Serialize(value)
	key := cipher.SumSHA256(data)

	if db.has(key) {
		return cipher.SHA256{}, fmt.Errorf("key already present: %v", key)
	}
	db.mu.Lock()
	db.data[key] = data
	db.mu.Unlock()
	//db.newDataCallback(key, value)
	return key, nil
}

func (db *Store) Load(key cipher.SHA256, data interface{}) error {
	value, ok := db.data[key]
	if !ok {
		return errors.New("Object does not exist")
	}
	encoder.DeserializeRaw(value, data)
	return nil
}

func (db *Store) Get(key cipher.SHA256) ([]byte, bool) {
	value, ok := db.data[key]
	return value, ok
}

func (db *Store) has(key cipher.SHA256) bool {
	_, ok := db.data[key]
	return ok
}

//func (db *Store) NewDataCallback(newDataCallback func(cipher.SHA256, interface{}) error) error {
//	db.mu.Lock()
//	defer db.mu.Unlock()
//
//	if newDataCallback != nil {
//		db.newDataCallback = newDataCallback
//	}
//	return nil
//}
