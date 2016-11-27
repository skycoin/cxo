package schema

import (
	"fmt"
	"sync"
	"errors"
	"github.com/skycoin/cxo/encoder"
)

type Store struct {
	data            map[HKey][]byte
	mu              *sync.RWMutex
	//newDataCallback func(cipher.SHA256, interface{}) error
}

func NewStore() *Store {
	db := Store{}
	db.data = make(map[HKey][]byte)
	db.mu = &sync.RWMutex{}
	return &db
}

func (db *Store) saveObj(value interface{}) (HKey, error) {
	data := encoder.Serialize(value)
	key := CreateKey(data)

	if db.Has(key) {
		return key, fmt.Errorf("key already present: %v", key)
	}
	db.mu.Lock()
	db.data[key] = data
	db.mu.Unlock()
	//db.newDataCallback(key, value)
	return key, nil
}

func (db *Store) Add(key HKey, data []byte) {
	db.mu.Lock()
	db.data[key] = data
	db.mu.Unlock()
}

func (db *Store) Save(value interface{}) (Href, error) {
	key, e := db.saveObj(value)
	if (e != nil) {
		return Href{}, e
	}
	return NewHref(key, value), nil
}


func (db *Store) Load(key HKey, data interface{}) error {
	value, ok := db.data[key]
	if !ok {
		return errors.New("Object does not exist")
	}
	encoder.DeserializeRaw(value, data)
	return nil
}

func (db *Store) Get(key HKey) ([]byte, bool) {
	value, ok := db.data[key]
	return value, ok
}

func (db *Store) Has(key HKey) bool {
	_, ok := db.data[key]
	return ok
}

func (db *Store) CreateArray(objType interface{}, items ...HKey) HArray{
	schema := ExtractSchema(objType)
	key, _ := db.saveObj(schema)
	return newArray(key, items...)
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
