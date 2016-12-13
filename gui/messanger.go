package gui

import (
	"github.com/skycoin/skycoin/src/cipher"
	"fmt"
)

type Messenger struct {
	Context interface{}
}

type IMessageTransport interface {
	Announce(hash cipher.SHA256) error
	Request(hash cipher.SHA256) error
	Data(hash cipher.SHA256, data []byte) error
}

func (m *Messenger) Announce(hash cipher.SHA256) error {
	fmt.Println(m.Context)
	return m.Context.(IMessageTransport).Announce(hash)
}

func (m *Messenger) Request(hash cipher.SHA256) error {
	return m.Context.(IMessageTransport).Request(hash)
}

func (m *Messenger) Data(hash cipher.SHA256, data []byte)  error{
	return m.Context.(IMessageTransport).Data(hash, data)
}

