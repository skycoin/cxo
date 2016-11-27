package gui

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/skycoin/skycoin/src/cipher"
)

func (ctx *Context) Param(key string) *string {
	if key == "" {
		return nil
	}
	value, ok := ctx.parameters[key]
	if !ok {
		return nil
	}
	return &value
}

func (ctx *Context) PubKeyFromParam(key string) (*cipher.PubKey, error) {
	if key == "" {
		return nil, errors.New("Empty key")
	}
	value, ok := ctx.parameters[key]
	if !ok {
		return nil, errors.New("Param not found")
	}
	logger.Debugf("value: %v", value)

	pubKey, err := cipher.PubKeyFromHex(value)
	if err != nil {
		return nil, err
	}
	logger.Debugf("pubKey: %v", pubKey)

	return &pubKey, nil
}

func (ctx *Context) JSON(code int, v interface{}) error {
	js, err := json.Marshal(v)
	if err != nil {
		ctx.ErrInternal(ErrorInternal.Error())
		return err
	}

	ctx.Response.Header().Set("Content-Type", "application/json")
	return ctx.write(code, js)
}

func (ctx *Context) write(code int, v []byte) error {
	_, err := ctx.Response.Write(v)
	ctx.Response.WriteHeader(code)
	return err
}

func (ctx *Context) Text(code int, v string) error {
	ctx.Response.WriteHeader(code)
	_, err := io.WriteString(ctx.Response, v)
	return err
}
