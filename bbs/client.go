package bbs

type client struct{
	bbs *Bbs
}

func NewClient() *client{
	return &client{}
}

