package node

import (
	"encoding/json"

	"github.com/skycoin/skycoin/src/cipher"
)

// A Connection represents incoming or outgoing connection:
// its public key and address. The address can be an IPv6 address.
// The Connection contains pure pubic key but json-encoded value
// contain hexadecimal-encode string. See MarshalJSON for details
type Connection struct {
	Pub  cipher.PubKey `json:"pub"`
	Addr string        `json:"addr"`
}

// MarshalJSON inmplements encoding/json.Marshaler interface.
// It encodes Connection to json object, in which the Pub field
// (public key) represented by hexadecimal-encoded string
// (instead of array of numbers). For example:
//
//     {
//         "pub": "020d4a52d13c4d218d1d053b9e5f09015b6dfd88bec92e3c2462b59a4a6d34c8bb",
//         "addr": "127.0.0.1:6482"
//     }
//
// Be careful. The MarshalJSON method has a pointer receiver
// but if you pass an unaddressabel value to
// (encoding/json).Marshal such as Connection{someAddr, someKey}
// then the method will not be used and encoded value will contain
// array of integers instead of hexadecimal-encoded string
func (c *Connection) MarshalJSON() (data []byte, _ error) {
	// hex + {"pub":"","addr":""} + addr
	data = make([]byte, 0, 66+20+20) // scratch
	data = append(data, `{"pub":"`...)
	data = append(data, c.Pub.Hex()...)
	data = append(data, `","addr":"`...)
	data = append(data, c.Addr...)
	data = append(data, `"}`...)
	return
}

// UnmarshalJOSN implements (encoding/json).Unmarshaler interface
func (c *Connection) UnmarshalJSON(data []byte) (err error) {
	var temp struct {
		Pub  string
		Addr string
	}
	if err = json.Unmarshal(data, &temp); err != nil {
		return
	}
	if c.Pub, err = cipher.PubKeyFromHex(temp.Pub); err != nil {
		return
	}
	c.Addr = temp.Addr
	return
}
