package data

import (
	"fmt"
	"sync"
	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/cxo/encoder"
)

type DataBase struct {
	data            map[cipher.SHA256][]byte
	mu              *sync.RWMutex
	newDataCallback func(cipher.SHA256, interface{}) error
}

type Statistic struct {
	Total  int        `json:"total"`
	Memory int        `json:"memory"`
}

type queryCondition func(key cipher.SHA256, data []byte) bool

type IDataSource interface {
	Save(value interface{}) cipher.SHA256
	Update(value []byte) cipher.SHA256
	Add(ds cipher.SHA256, value []byte) error
	Has(ds cipher.SHA256) bool
	Get(ds cipher.SHA256) ([]byte, bool)
	Where(queryCondition) []cipher.SHA256
	Statistic() *Statistic

	GetData() map[cipher.SHA256][]byte
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

func createKey(data []byte) cipher.SHA256 {
	return cipher.SumSHA256(data)
}

func (db *DataBase) Save(value interface{}) cipher.SHA256 {
	data := encoder.Serialize(value)
	return db.Update(data)
}

func (db *DataBase) Update(data []byte) cipher.SHA256 {
	key := createKey(data)
	if (key == cipher.SHA256{} || data == nil ) {
		panic("Invalid key")
	}
	//fmt.Println("add", key, data)
	db.mu.Lock()
	db.data[key] = data
	db.mu.Unlock()
	return key
}

func (db *DataBase) Add(key cipher.SHA256, value []byte) error {
	if (key == cipher.SHA256{} || value == nil ) {
		panic("Invalid key")
	}

	if db.Has(key) {
		return fmt.Errorf("key already present: %v", key)
	}

	//fmt.Println("add", key, value)
	db.mu.Lock()
	db.data[key] = value
	db.mu.Unlock()

	if (db.newDataCallback != nil) {
		db.newDataCallback(key, value)
	}
	return nil
}

func (db *DataBase) Has(key cipher.SHA256) bool {
	db.mu.Lock()
	_, ok := db.data[key]
	db.mu.Unlock()
	return ok
}

func (db *DataBase) Get(key cipher.SHA256) ([]byte, bool) {
	db.mu.Lock()
	value, ok := db.data[key]
	db.mu.Unlock()
	return value, ok
}

func (db *DataBase) Where(q queryCondition) []cipher.SHA256 {
	result := []cipher.SHA256{}

	for key := range db.data {

		if (q(key, db.data[key])) {
			result = append(result, key)
		}
	}
	return result
}

func (db *DataBase) Statistic() *Statistic {
	res := &Statistic{Total:len(db.data)}
	for i := 0; i < res.Total; i++ {
		res.Memory += len(db.data)
	}
	return res
}

func (db *DataBase) GetData() map[cipher.SHA256][]byte {
	return db.data
}
