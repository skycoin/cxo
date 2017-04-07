package node

import (
	"github.com/skycoin/cxo/node/gnet"
	"github.com/skycoin/skycoin/src/cipher"
)

func registerMessages(p *gnet.Pool) {
	// data
	p.Register(gnet.NewPrefix("ANNC"), &Announce{})
	p.Register(gnet.NewPrefix("RQST"), &Request{})
	p.Register(gnet.NewPrefix("DATA"), &Data{})

	// root
	p.Register(gnet.NewPrefix("ANRT"), &AnnounceRoot{})
	p.Register(gnet.NewPrefix("RQRT"), &RequestRoot{})
	p.Register(gnet.NewPrefix("DTRT"), &DataRoot{})
}

//
// data
//

type Announce struct {
	Hash cipher.SHA256
}

type Request struct {
	Hash cipher.SHA256
}

type Data struct {
	Data []byte
}

//
// Root
//

type AnnounceRoot struct {
	Pub  cipher.PubKey
	Time int64
}

type RequestRoot struct {
	Pub  cipher.PubKey
	Time int64
}

type DataRoot struct {
	Pub  cipher.PubKey
	Sig  cipher.Sig
	Root []byte
}
