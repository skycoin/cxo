package registry

import (
	"testing"
	//"github.com/skycoin/skycoin/src/cipher"
)

func TestRefs_String(t *testing.T) {
	// String() string

	//

}

func TestRefs_Short(t *testing.T) {
	// Short() string

	//

}

func TestRefs_Init(t *testing.T) {
	// Init(pack Pack) (err error)

	//

}

func TestRefs_Len(t *testing.T) {
	// Len(pack Pack) (ln int, err error)

	//

}

func TestRefs_Depth(t *testing.T) {
	// Depth(pack Pack) (depth int, err error)

	//

}

func TestRefs_Degree(t *testing.T) {
	// Degree(pack Pack) (degree int, err error)

	//

}

func TestRefs_Flags(t *testing.T) {
	// Flags() (flags Flags)

	//

}

func TestRefs_Reset(t *testing.T) {
	// Reset() (err error)

	//

}

func TestRefs_HasHash(t *testing.T) {
	// HasHsah(pack Pack, hash cipher.SHA256) (ok bool, err error)

	//

}

func TestRefs_ValueByHash(t *testing.T) {
	// ValueByHash(pack Pack, hash cipher.SHA256, obj interface{}) (err error)

	//

}

func TestRefs_IndexOfHash(t *testing.T) {
	// IndexOfHash(pack Pack, hash cipher.SHA256) (i int, err error)

	//

}

func TestRefs_IndicesByHash(t *testing.T) {
	// IndicesByHash(pack Pack, hash cipher.SHA256) (is []int, err error)

	//

}

func TestRefs_ValueOfHashWithIndex(t *testing.T) {
	// ValueOfHashWithIndex(pack Pack, hash cipher.SHA256,
	//     obj interface{}) (i int, err error)

	//

}

func TestRefs_HashByIndex(t *testing.T) {
	// HashByIndex(pack Pack, i int) (hash cipher.SHA256, err error)

	//

}

func TestRefs_ValueByIndex(t *testing.T) {
	// ValueByIndex(pack Pack, i int, obj interface{}) (hash cipher.SHA256,
	//     err error)

	//

}

func TestRefs_SetHashByIndex(t *testing.T) {
	// SetHashByIndex(pack Pack, i int, hash cipher.SHA256) (err error)

	//

}

func TestRefs_SetValueByIndex(t *testing.T) {
	// SetValueByIndex(pack Pack, i int, obj interface{}) (err error)

	//

}

func TestRefs_DeleteByIndex(t *testing.T) {
	// DeleteByIndex(pack Pack, i int) (err error)

	//

}

func TestRefs_DeleteByHash(t *testing.T) {
	// DeleteByHash(pack Pack, hash cipher.SHA256) (err error)

	//

}

func TestRefs_Ascend(t *testing.T) {
	// Ascend(pack Pack, ascendFunc IterateFunc) (err error)

	//

}

func TestRefs_AscendFrom(t *testing.T) {
	// AscendFrom(pack Pack, from int, ascendFunc IterateFunc) (err error)

	//
}

func TestRefs_Descend(t *testing.T) {
	// Descend(pack Pack, descendFunc IterateFunc) (err error)

	//

}

func TestRefs_DescendFrom(t *testing.T) {
	// DescendFrom(pack Pack, from int, descendFunc IterateFunc) (err error)

	//

}

func TestRefs_Slice(t *testing.T) {
	// Slice(pack Pack, i int, j int) (slice *Refs, err error)

	//

}

func TestRefs_Append(t *testing.T) {
	// Append(pack Pack, refs *Refs) (err error)

	//

}

func TestRefs_AppendValues(t *testing.T) {
	// AppendValues(pack Pack, values ...interface{}) (err error)

	//

}

func TestRefs_AppendHashes(t *testing.T) {
	// AppendHashes(pack Pack, hashes ...cipher.SHA256) (err error)

	//

}

func TestRefs_Clear(t *testing.T) {
	// Clear()

	//

}

func TestRefs_Rebuild(t *testing.T) {
	// Rebuild(pack Pack) (err error)

	//

}

func TestRefs_Tree(t *testing.T) {
	// Tree() (tree string)

	//

}
