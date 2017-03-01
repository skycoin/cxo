package main

import (
	"github.com/skycoin/skycoin/src/cipher"
)

// Singer implements the bbs.Signer interface.
type Signer struct {
}

// Sign member.
func (s Signer) Sign(hash cipher.SHA256) cipher.Sig {
	sig, _ := cipher.SigFromHex(hash.Hex())
	return sig
}
