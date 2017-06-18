package node

import (
	"testing"
	"time"

	"github.com/skycoin/skycoin/src/cipher"

	"github.com/skycoin/cxo/skyobject"
)

/*

$ go test -bench . -benchtime 10s
BenchmarkNode_srcDst/empty_roots_memory-4       10000    2390448 ns/op  100237 B/op  1690 allocs/op
BenchmarkNode_srcDst/empty_roots_boltdb-4         100  113579500 ns/op  135854 B/op  1911 allocs/op
BenchmarkNode_passThrough/empty_roots_memory-4   5000    3082852 ns/op  128090 B/op  2159 allocs/op
BenchmarkNode_passThrough/empty_roots_boltdb-4    100  173033312 ns/op  184738 B/op  2549 allocs/op
PASS
ok      github.com/logrusorgru/cxo/node 73.241s

*/

//
// TODO: DRY
//

func benchmarkNodeSrcDstEmptyRootsMemory(b *testing.B) {

	// prepare
	reg := skyobject.NewRegistry()
	reg.Register("bench.User", User{})

	sconf := newConfig(false)
	dconf := newConfig(true)

	filled := make(chan struct{})
	dconf.OnRootFilled = func(*Root) {
		filled <- struct{}{}
	}

	s, d, sc, _, err := newConnectedNodes(sconf, dconf)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	defer d.Close()

	pk, sk := cipher.GenerateKeyPair()

	d.Subscribe(nil, pk)
	if err := s.SubscribeResponse(sc, pk); err != nil {
		b.Error(err)
		return
	}

	// add registry and create first root
	cnt := s.Container()

	var root *Root

	// becasu we're not using "core registry" we need be sure that
	// GC doesn't cleans our registry until we create first root object
	cnt.LockGC()
	{
		cnt.AddRegistry(reg)
		root, err = cnt.NewRootReg(pk, sk, reg.Reference())
		if err != nil {
			cnt.UnlockGC()
			b.Error(err)
			return
		}
		if _, err = root.Touch(); err != nil { // crete first root
			cnt.UnlockGC()
			b.Error(err)
			return
		}

	}
	cnt.UnlockGC()

	select {
	case <-filled:
	case <-time.After(TM * 10):
		b.Error("slow or missing")
		return
	}

	// run
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// update
		if _, err = root.Touch(); err != nil {
			b.Error(err)
			return
		}
		// receive filled
		<-filled
	}

	// fini
	b.ReportAllocs()
}

func benchmarkNodeSrcDstEmptyRootsBoltDB(b *testing.B) {

	defer clean()

	// prepare
	reg := skyobject.NewRegistry()

	reg.Register("bench.User", User{})

	sconf := newConfig(false)
	sconf.InMemoryDB = false
	sconf.DBPath = sconf.DBPath + ".source"

	dconf := newConfig(true)
	dconf.InMemoryDB = false
	dconf.DBPath = dconf.DBPath + ".destination"

	filled := make(chan struct{})
	dconf.OnRootFilled = func(*Root) {
		filled <- struct{}{}
	}

	s, d, sc, _, err := newConnectedNodes(sconf, dconf)
	if err != nil {
		b.Fatal(err)
	}
	defer s.Close()
	defer d.Close()

	pk, sk := cipher.GenerateKeyPair()

	d.Subscribe(nil, pk)
	if err := s.SubscribeResponse(sc, pk); err != nil {
		b.Error(err)
		return
	}

	// add registry and create first root
	cnt := s.Container()

	var root *Root

	// becasu we're not using "core registry" we need be sure that
	// GC doesn't cleans our registry until we create first root object
	cnt.LockGC()
	{
		cnt.AddRegistry(reg)
		root, err = cnt.NewRootReg(pk, sk, reg.Reference())
		if err != nil {
			cnt.UnlockGC()
			b.Error(err)
			return
		}
		if _, err = root.Touch(); err != nil { // crete first root
			cnt.UnlockGC()
			b.Error(err)
			return
		}

	}
	cnt.UnlockGC()

	select {
	case <-filled:
	case <-time.After(TM * 10):
		b.Error("slow or missing")
		return
	}

	// run
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// update
		if _, err = root.Touch(); err != nil {
			b.Error(err)
			return
		}
		// receive filled
		<-filled
	}

	// fini
	b.ReportAllocs()
}

func BenchmarkNode_srcDst(b *testing.B) {

	b.Run("empty roots memory", benchmarkNodeSrcDstEmptyRootsMemory)
	b.Run("empty roots boltdb", benchmarkNodeSrcDstEmptyRootsBoltDB)

}

func benchmarkNodePassThroughEmptyRootsMemory(b *testing.B) {

	// prepare

	reg := skyobject.NewRegistry()
	reg.Register("bench.User", User{})

	sconf := newConfig(false)
	pconf := newConfig(true)
	dconf := newConfig(false)

	filled := make(chan struct{})
	dconf.OnRootFilled = func(*Root) {
		filled <- struct{}{}
	}

	// source
	src, err := NewNode(sconf)
	if err != nil {
		b.Error(err)
		return
	}
	defer src.Close()
	// pipe
	pipe, err := NewNode(pconf)
	if err != nil {
		b.Error(err)
		return
	}
	defer pipe.Close()
	// destination
	dst, err := NewNode(dconf)
	if err != nil {
		b.Error(err)
		return
	}
	defer dst.Close()

	// source->pipe
	sc, err := src.Pool().Dial(pipe.Pool().Address())
	if err != nil {
		b.Error(err)
		return
	}
	// destination->pipe
	dc, err := dst.Pool().Dial(pipe.Pool().Address())
	if err != nil {
		b.Error(err)
		return
	}

	pk, sk := cipher.GenerateKeyPair()

	// subcribe
	pipe.Subscribe(nil, pk)
	if err := src.SubscribeResponse(sc, pk); err != nil {
		b.Error(err)
		return
	}
	if err := dst.SubscribeResponse(dc, pk); err != nil {
		b.Error(err)
		return
	}

	// add registry and create first root
	cnt := src.Container()

	var root *Root

	// becasu we're not using "core registry" we need be sure that
	// GC doesn't cleans our registry until we create first root object
	cnt.LockGC()
	{
		cnt.AddRegistry(reg)
		root, err = cnt.NewRootReg(pk, sk, reg.Reference())
		if err != nil {
			cnt.UnlockGC()
			b.Error(err)
			return
		}
		if _, err = root.Touch(); err != nil { // crete first root
			cnt.UnlockGC()
			b.Error(err)
			return
		}

	}
	cnt.UnlockGC()

	select {
	case <-filled:
	case <-time.After(TM * 10):
		b.Error("slow or missing")
		return
	}

	// run
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// update
		if _, err = root.Touch(); err != nil {
			b.Error(err)
			return
		}
		// receive filled
		<-filled
	}

	// fini
	b.ReportAllocs()
}

func benchmarkNodePassThroughEmptyRootsBoltDB(b *testing.B) {

	defer clean()

	// prepare

	reg := skyobject.NewRegistry()
	reg.Register("bench.User", User{})

	sconf := newConfig(false)
	sconf.InMemoryDB = false
	sconf.DBPath = sconf.DBPath + ".source"
	pconf := newConfig(true)
	pconf.InMemoryDB = false
	pconf.DBPath = pconf.DBPath + ".pipe"
	dconf := newConfig(false)
	dconf.InMemoryDB = false
	dconf.DBPath = dconf.DBPath + ".destination"

	filled := make(chan struct{})
	dconf.OnRootFilled = func(*Root) {
		filled <- struct{}{}
	}

	// source
	src, err := NewNode(sconf)
	if err != nil {
		b.Error(err)
		return
	}
	defer src.Close()
	// pipe
	pipe, err := NewNode(pconf)
	if err != nil {
		b.Error(err)
		return
	}
	defer pipe.Close()
	// destination
	dst, err := NewNode(dconf)
	if err != nil {
		b.Error(err)
		return
	}
	defer dst.Close()

	// source->pipe
	sc, err := src.Pool().Dial(pipe.Pool().Address())
	if err != nil {
		b.Error(err)
		return
	}
	// destination->pipe
	dc, err := dst.Pool().Dial(pipe.Pool().Address())
	if err != nil {
		b.Error(err)
		return
	}

	pk, sk := cipher.GenerateKeyPair()

	// subcribe
	pipe.Subscribe(nil, pk)
	if err := src.SubscribeResponse(sc, pk); err != nil {
		b.Error(err)
		return
	}
	if err := dst.SubscribeResponse(dc, pk); err != nil {
		b.Error(err)
		return
	}

	// add registry and create first root
	cnt := src.Container()

	var root *Root

	// becasu we're not using "core registry" we need be sure that
	// GC doesn't cleans our registry until we create first root object
	cnt.LockGC()
	{
		cnt.AddRegistry(reg)
		root, err = cnt.NewRootReg(pk, sk, reg.Reference())
		if err != nil {
			cnt.UnlockGC()
			b.Error(err)
			return
		}
		if _, err = root.Touch(); err != nil { // crete first root
			cnt.UnlockGC()
			b.Error(err)
			return
		}

	}
	cnt.UnlockGC()

	select {
	case <-filled:
	case <-time.After(TM * 10):
		b.Error("slow or missing")
		return
	}

	// run
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// update
		if _, err = root.Touch(); err != nil {
			b.Error(err)
			return
		}
		// receive filled
		<-filled
	}

	// fini
	b.ReportAllocs()
}

func BenchmarkNode_passThrough(b *testing.B) {

	b.Run("empty roots memory", benchmarkNodePassThroughEmptyRootsMemory)
	b.Run("empty roots boltdb", benchmarkNodePassThroughEmptyRootsBoltDB)

}
