package gui

import (
	"github.com/skycoin/skycoin/src/cipher"
)

type Messenger struct {
	Context interface{}
}

type IMessageTransport interface {
	Announce(hash cipher.SHA256) error
	Request(hash cipher.SHA256) error
}

func (m *Messenger) Announce(hash cipher.SHA256) error {

	return m.Context.(IMessageTransport).Announce(hash)
}

func (m *Messenger) Request(hash cipher.SHA256) error {
	return m.Context.(IMessageTransport).Request(hash)
}
