package skyobject

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	c := NewConfig()
	for _, degree := range []int{-1, 0, 1} {
		c.MerkleDegree = degree
		if c.Validate() == nil {
			t.Error("missing error")
		}
	}
	c.MerkleDegree = 2
	if err := c.Validate(); err != nil {
		t.Error("unexpected error:", err)
	}
}
