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

// A User represents brief
// information about a user
type User struct {
	Name string
}

// A Tweet represents
// message with header,
// body and time
type Tweet struct {
	Head string
	Body string
	Time int64 // unix nano
}

// A Feed represents list
// of tweets of a User and
// the user info
type Feed struct {
	User   cxo.Ref  `skyobject:"schema=tw.User"`
	Tweets cxo.Refs `skyobject:"schema=tw.Tweet"`
}
