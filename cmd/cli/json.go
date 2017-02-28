package main

import (
	"encoding/json"
	"fmt"
	"io"
)

func readResponse(resp io.Reader) (jr JSONResponse, err error) {
	err = json.NewDecoder(resp).Decode(&jr)
	if err != nil {
		err = fmt.Errorf("error decoding response: %v", err)
	}
	return
}

// nessesary JSON-structures

// skycoin/cxo/gui/errors.go
type JSONResponse struct {
	Code   string                  `json:"code,omitempty"`
	Status int                     `json:"status,omitempty"`
	Detail string                  `json:"detail,omitempty"`
	Meta   *map[string]interface{} `json:"meta,omitempty"`
}

// list nodes or list subscriptions
type Item struct {
	Address string `json:"addr"`
	PubKey  string `json:"pub"`
}

// stat cxo/data/db.go
type Statistic struct {
	Total  int `json:"total"`
	Memory int `json:"memory"`
}

type NodeInfo struct {
	Address   string `json:"address"`
	Listening bool   `json:"listening"`
	PubKey    string `json:"pubKey"`
}
