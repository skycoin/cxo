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
