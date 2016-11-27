package main

import (
	"fmt"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
)

type DataBase struct {
	data            map[cipher.SHA256][]byte
	mu              *sync.RWMutex
	newDataCallback func(cipher.SHA256, interface{}) error
}

func NewDB() *DataBase {
	db := DataBase{}
	db.data = make(map[cipher.SHA256][]byte)
	db.mu = &sync.RWMutex{}
	return &db
}

func (db *DataBase) NewDataCallback(newDataCallback func(cipher.SHA256, interface{}) error) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if newDataCallback != nil {
		db.newDataCallback = newDataCallback
	}
	return nil
}

func (db *DataBase) Add(key cipher.SHA256, value []byte) error {

	if db.Has(key) {
		return fmt.Errorf("key already present: %v", key)
	}
	db.mu.Lock()
	db.data[key] = value
	db.mu.Unlock()

	db.newDataCallback(key, value)
	return nil
}

func (db *DataBase) Has(key cipher.SHA256) bool {
	_, ok := db.data[key]
	return ok
}

func (db *DataBase) Get(key cipher.SHA256) ([]byte, bool) {
	value, ok := db.data[key]
	return value, ok
}
