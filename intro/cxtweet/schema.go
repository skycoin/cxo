package main

import (
	cxo "github.com/skycoin/cxo/skyobject"
)

// CXO Feed -> CXO Root -> cxtweet.Feed
//
// cxtweet.Feed
//  |
//  +-> cxtweet.User
//  |
//  +-> cxtweet.Tweets (that is []cxtweet.Tweet)
//       |
//       +-> first tweet
//       +-> second tweet
//       +-> and so on

type User struct {
	Name string
}

type Tweet struct {
	Head string
	Body string
	Time int64 // unix nano
}

type Feed struct {
	User   cxo.Ref  `skyobject:"schema=tw.User"`
	Tweets cxo.Refs `skyobject:"schema=tw.Tweet"`
}
