package data

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
)

func TestStat_String(t *testing.T) {
	s := Stat{}
	if got, want := s.String(), "{total: 0, memory: 0B}"; want != got {
		t.Error("want %q, got %q", want, got)
	}
}

func TestNewDB(t *testing.T) {
	d := NewDB()
	if d == nil {
		t.Error("NewDb returns nil")
	}
	if d.data == nil {
		t.Error("nil map")
	}
}

func TestDB_Has(t *testing.T) {
	d := NewDB()
	data := []byte("x")
	key := cipher.SumSHA256(data)
	if d.Has(key) {
		t.Error("has unexisted value")
	}
	d.Set(key, data)
	if !d.Has(key) {
		t.Error("hasn't existed value")
	}
}

func TestDB_Get(t *testing.T) {
	d := NewDB()
	data := []byte("x")
	key := cipher.SumSHA256(data)
	if _, ok := d.Get(key); ok {
		t.Error("got unexisted value")
	}
	d.Set(key, data)
	if v, ok := d.Get(key); !ok {
		t.Error("can't get existed value")
	} else if string(v) != string(data) {
		t.Error("wrong value")
	}
}

func TestDB_Set(t *testing.T) {
	d := NewDB()
	data := []byte("x")
	key := cipher.SumSHA256(data)
	d.Set(key, data)
	if v, ok := d.data[key]; !ok {
		t.Error("can't get existed value")
	} else if string(v) != string(data) {
		t.Error("wrong value")
	}
	d.Set(key, data) // overwrite
	if v, ok := d.data[key]; !ok {
		t.Error("can't get existed value")
	} else if string(v) != string(data) {
		t.Error("wrong value")
	}
}

func TestDB_Range(t *testing.T) {
	d := NewDB()
	for _, v := range []string{"a", "b", "c"} {
		data := []byte(v)
		key := cipher.SumSHA256(data)
		d.Set(key, data)
	}
	got := []string{}
	d.Range(func(key cipher.SHA256) {
		val, ok := d.Get(key)
		if !ok {
			t.Error("missing data")
			return
		}
		got = append(got, string(val))
	})
	if len(got) != 3 {
		t.Error("wrong range")
	}
}

func TestDB_Stat(t *testing.T) {
	d := NewDB()
	if d.Stat() != (Stat{}) {
		t.Error("wrong stat")
	}
	data := []byte("x")
	key := cipher.SumSHA256(data)
	d.Set(key, data)
	if s := d.Stat(); s.Total != 1 {
		t.Error("missmatch total")
	} else if s.Memory != len(data) {
		t.Error("missmatch memory")
	}
}

func TestDB_AddAutoKey(t *testing.T) {
	d := NewDB()
	data := []byte("x")
	key := cipher.SumSHA256(data)
	k2 := d.AddAutoKey(data)
	if k2 != key {
		t.Error("wrong auto key")
		return
	}
	if v, ok := d.data[key]; !ok {
		t.Error("can't get existed value")
	} else if string(v) != string(data) {
		t.Error("wrong value")
	}
}
