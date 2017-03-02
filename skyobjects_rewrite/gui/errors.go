package gui

import (
	"errors"
	"fmt"
)

var ErrorInvalidRequest = errors.New("request not valid")
var ErrorInternal = errors.New("internal server error")
var ErrorNotFound = errors.New("not found")
var ErrorParamNotFound = errors.New("param not found")

type JSONResponse struct {
	Code   string                  `json:"code,omitempty"`
	Status int                     `json:"status,omitempty"`
	Detail string                  `json:"detail,omitempty"`
	Meta   *map[string]interface{} `json:"meta,omitempty"`
}

func (ctx *Context) ErrInvalidRequest(message string, keyvals ...interface{}) error {
	errorResponse := JSONResponse{
		Code:   "invalid request",
		Status: 400,
		Detail: message,
	}
	if len(keyvals) > 0 {
		meta := keyVals(keyvals...)
		errorResponse.Meta = &meta
	}
	return ctx.JSON(400, errorResponse)
}
func (ctx *Context) ErrInternal(message string, keyvals ...interface{}) error {
	errorResponse := JSONResponse{
		Code:   "internal",
		Status: 500,
		Detail: message,
	}
	if len(keyvals) > 0 {
		meta := keyVals(keyvals...)
		errorResponse.Meta = &meta
	}
	return ctx.JSON(500, errorResponse)
}
func (ctx *Context) ErrNotFound(message string, keyvals ...interface{}) error {
	errorResponse := JSONResponse{
		Code:   "not found",
		Status: 404,
		Detail: message,
	}
	if len(keyvals) > 0 {
		meta := keyVals(keyvals...)
		errorResponse.Meta = &meta
	}
	return ctx.JSON(404, errorResponse)
}

func keyVals(keyvals ...interface{}) map[string]interface{} {
	if len(keyvals) == 0 {
		return nil
	}
	meta := make(map[string]interface{}, (len(keyvals)+1)/2)
	for i := 0; i < len(keyvals); i += 2 {
		k := keyvals[i]
		//fmt.Println("i", i)
		//fmt.Println("k", k)
		var v interface{} = "MISSING"
		if i+1 < len(keyvals) {
			v = keyvals[i+1]
		}
		//fmt.Println("v", v)
		meta[fmt.Sprint(k)] = v
	}
	return meta
}
