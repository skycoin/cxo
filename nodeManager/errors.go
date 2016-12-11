package nodeManager

import "errors"

var ErrPubKeyIsNil = errors.New("pubKey is nil")
var ErrHandshakeTimeout = errors.New("handshake timeout")
