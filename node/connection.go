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
// (instead of base64 encoded). For example:
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
