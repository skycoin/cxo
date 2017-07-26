package skyobject

type WalkerNode struct {
	Root *Root

	value     interface{}
	isChanged bool
}
