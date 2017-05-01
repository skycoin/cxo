package main

import (
	"fmt"

	"github.com/skycoin/skycoin/src/cipher"
)

func main() {
	pk, sk := cipher.GenerateKeyPair()
	fmt.Println("pk:", pk.Hex())
	fmt.Println("sk:", sk.Hex())
}
