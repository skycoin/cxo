package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/skycoin/cxo/intro/discovery/discovery/db"
	"github.com/skycoin/net/skycoin-messenger/factory"
)

const (
	Address string = ":8080"
)

func main() {

	var address = Address

	flag.StringVar(&address,
		"a",
		address,
		"listening address")
	flag.Parse()

	var m = newDiscovery()

	// initialize SQLite3 DB
	if err := db.Init(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// start discovery listener
	if err := m.Listen(address); err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	waitInterrupt() // wait for SIGINT
}

func waitInterrupt() {
	var sig = make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func newDiscovery() (m *factory.MessengerFactory) {
	m = factory.NewMessengerFactory()

	// use random seed every start
	if err := m.SetDefaultSeedConfig(factory.NewSeedConfig()); err != nil {
		panic(err)
	}

	// use SQLite3 DB to keep information in
	m.RegisterService = db.RegisterService
	m.UnRegisterService = db.UnRegisterService
	m.FindByAttributes = db.FindResultByAttrs
	m.FindByAttributesAndPaging = db.FindResultByAttrsAndPaging
	m.FindServiceAddresses = db.FindServiceAddresses

	return m
}
