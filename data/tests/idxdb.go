package tests

import (
	"testing"

	"github.com/skycoin/cxo/data"
)

func IdxDBClose(t *testing.T, idx data.IdxDB) {
	if err := idx.Close(); err != nil {
		t.Error(err)
	}
	if err := idx.Close(); err != nil {
		t.Error(err)
	}
}
