package gui

import (
	"encoding/json"
	"errors"
	"io"
	"strconv"

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

func (ctx *Context) SHA256FromParam(key string) (*cipher.SHA256, error) {
	if key == "" {
		return nil, errors.New("Empty key")
	}
	value, ok := ctx.parameters[key]
	if !ok {
		return nil, ErrorParamNotFound
	}
	//logger.Debugf("value: %v", value)

	SHA256, err := cipher.SHA256FromHex(value)
	if err != nil {
		return nil, err
	}
	//logger.Debugf("SHA256: %v", SHA256.Hex())

	return &SHA256, nil
}

func (ctx *Context) IntFromParam(key string) (*int, error) {
	if key == "" {
		return nil, errors.New("Empty key")
	}
	value, ok := ctx.parameters[key]
	if !ok {
		return nil, ErrorParamNotFound
	}
	//logger.Debugf("value: %v", value)

	integer, err := strconv.Atoi(value)
	if err != nil {
		return nil, err
	}
	//logger.Debugf("integer: %v", integer)

	return &integer, nil
}

func (ctx *Context) PubKeyFromParam(key string) (*cipher.PubKey, error) {
	if key == "" {
		return nil, errors.New("Empty key")
	}
	value, ok := ctx.parameters[key]
	if !ok {
		return nil, ErrorParamNotFound
	}
	//logger.Debugf("value: %v", value)

	pubKey, err := cipher.PubKeyFromHex(value)
	if err != nil {
		return nil, err
	}
	//logger.Debugf("pubKey: %v", pubKey.Hex())

	return &pubKey, nil
}

func (ctx *Context) JSON(code int, v interface{}) error {
	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(code)
	return json.NewEncoder(ctx.Response).Encode(v)
}

func (ctx *Context) Text(code int, v string) error {
	ctx.Response.Header().Set("Content-Type", "text/html")
	ctx.Response.WriteHeader(code)
	_, err := io.WriteString(ctx.Response, v)
	return err
}

func (ctx *Context) write(code int, v []byte) error {
	ctx.Response.WriteHeader(code)
	_, err := ctx.Response.Write(v)
	return err
}

func (ctx *Context) writeErrorJSON(code int, v interface{}) error {
	js, err := json.Marshal(v)
	if err != nil {
		ctx.ErrInternal(ErrorInternal.Error())
		return err
	}

	ctx.Response.Header().Set("Content-Type", "application/json")
	return ctx.write(code, js)
}
