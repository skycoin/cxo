package data

import (
	"sort"
	"sync"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"
)

// buckets:
//  - objects hash -> []byte (including schemas)
//  - feeds   pubkey -> { seq -> RootPack }
type memoryDB struct {
	mx      sync.RWMutex
	objects map[cipher.SHA256][]byte
	feeds   map[cipher.PubKey]map[uint64]RootPack
}

// NewMemoryDB creates new database in memory
//
// TODO (kostyarin): in-memory-db doesn't supprot transactions
// and all changes performed in place
func NewMemoryDB() (db DB) {
	db = &memoryDB{
		objects: make(map[cipher.SHA256][]byte),
		feeds:   make(map[cipher.PubKey]map[uint64]RootPack),
	}
	return
}

func (m *memoryDB) View(fn func(t Tv) error) (err error) {
	m.mx.RLock()
	defer m.mx.RUnlock()

	return fn(memoryTv{m})
}

func (m *memoryDB) Update(fn func(t Tu) error) (err error) {
	m.mx.Lock()
	defer m.mx.Unlock()

	// TODO: transactions

	return fn(memoryTu{m})
}

func (m *memoryDB) Stat() (s Stat) {

	m.mx.RLock()
	defer m.mx.RUnlock()

	// objects

	s.Objects = len(m.objects)

	for _, v := range m.objects {
		s.Space += Space(len(v))
	}

	// feeds (and roots)

	if len(m.feeds) == 0 {
		return // no feeds
	}

	s.Feeds = make(map[cipher.PubKey]FeedStat, len(m.feeds))

	emptyRootPackLen := len(encoder.Serialize(RootPack{}))

	for f, rs := range m.feeds {
		var fs FeedStat

		fs.Roots = len(rs)

		for _, r := range rs {
			fs.Space += r.Root + emptyRootPackLen
		}

		s.Feeds[cp] = fs

		return
	}

	return

}

func (m *memoryDB) Close() (err error) {
	// do nothing properly
	return
}

type driveTv struct {
	tx *bolt.Tx
}

func (d *driveTv) Objects() ViewObjects {
	o := new(driveObjects)
	o.bk = d.tx.Bucket(objectsBucket)
	return o
}

func (d *driveTv) Feeds() ViewFeeds {
	f := new(driveFeeds)
	f.bk = d.tx.Bucket(feedsBucket)
	return &driveViewFeeds{f}
}

type driveTu struct {
	tx *bolt.Tx
}

func (d *driveTu) Objects() UpdateObjects {
	o := new(driveObjects)
	o.bk = d.tx.Bucket(objectsBucket)
	return o
}

func (d *driveTu) Feeds() UpdateFeeds {
	f := new(driveFeeds)
	f.bk = d.tx.Bucket(feedsBucket)
	return f
}

type driveObjects struct {
	bk *bolt.Bucket
}

func (d *driveObjects) Set(key cipher.SHA256, value []byte) (err error) {
	return d.bk.Put(key[:], value)
}

func (d *driveObjects) Del(key cipher.SHA256) (err error) {
	return d.bk.Delete(key[:])
}

func (d *driveObjects) Get(key cipher.SHA256) []byte {
	return d.bk.Get(key[:])
}

func (d *driveObjects) GetCopy(key cipher.SHA256) (value []byte) {
	if g := d.bk.Get(key[:]); g != nil {
		value = make([]byte, len(g))
		copy(value, g)
	}
	return
}

func (d *driveObjects) Add(value []byte) (key cipher.SHA256, err error) {
	key = cipher.SumSHA256(value)
	err = d.bk.Put(key[:], value)
	return
}

func (d *driveObjects) IsExist(key cipher.SHA256) bool {
	return d.Get(key) != nil
}

func (d *driveObjects) SetMap(m map[cipher.SHA256][]byte) (err error) {
	for _, kv := range sortMap(m) {
		if err = d.bk.Put(kv.key[:], kv.val); err != nil {
			return
		}
	}
	return
}

func (d *driveObjects) Range(
	fn func(key cipher.SHA256, value []byte) error) (err error) {

	c := d.bk.Cursor()

	var ck cipher.SHA256

	for k, v := c.First(); k != nil; k, v = c.Next() {
		copy(ck[:], k)
		if err = fn(ck, v); err != nil {
			break
		}
	}

	if err == ErrStopRange {
		err = nil
	}
	return
}

func (d *driveObjects) RangeDel(
	fn func(key cipher.SHA256, value []byte) (bool, error)) (err error) {

	c := d.bk.Cursor()

	var ck cipher.SHA256
	var del bool

	// seek loop
	for k, v := c.First(); k != nil; k, v = c.Seek(k) {
		// next loop
		for {
			copy(ck[:], k)
			if del, err = fn(ck, v); err != nil {
				if err == ErrStopRange {
					err = nil
				}
				return
			}
			if del {
				if err = c.Delete(); err != nil {
					return
				}
				// coninue seek loop, because after deleting
				// we have got invalid cusor and we need to
				// call Seek to make it valid; the Seek will
				// points to next item, because current one
				// has been deleted
				break
			}
			// just get next item (next loop)
			if k, v = c.Next(); k == nil {
				return // there's nothing more
			}
		}
	}

	return
}

type driveFeeds struct {
	bk *bolt.Bucket
}

func (d *driveFeeds) Add(pk cipher.PubKey) (err error) {
	_, err = d.bk.CreateBucketIfNotExists(pk[:])
	return
}

func (d *driveFeeds) Del(pk cipher.PubKey) error {
	return d.bk.DeleteBucket(pk[:])
}

func (d *driveFeeds) IsExist(pk cipher.PubKey) bool {
	return d.bk.Bucket(pk[:]) != nil
}

func (d *driveFeeds) List() (list []cipher.PubKey) {

	ln := d.bk.Stats().KeyN
	if ln == 0 {
		return // nil
	}

	list = make([]cipher.PubKey, 0, ln)

	var cp cipher.PubKey
	c := d.bk.Cursor()

	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		copy(cp[:], k)
		list = append(list, cp)
	}

	return
}

func (d *driveFeeds) Range(fn func(pk cipher.PubKey) error) (err error) {
	var cp cipher.PubKey
	c := d.bk.Cursor()

	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		copy(cp[:], k)
		if err = fn(cp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

func (d *driveFeeds) RangeDel(
	fn func(pk cipher.PubKey) (bool, error)) (err error) {

	var cp cipher.PubKey
	var del bool

	c := d.bk.Cursor()

	// seek loop
	for k, _ := c.First(); k != nil; k, _ = c.Seek(k) {

		// next loop
		for {

			copy(cp[:], k)
			if del, err = fn(cp); err != nil {
				if err == ErrStopRange {
					err = nil
				}
				return
			}
			if del {
				if err = d.bk.DeleteBucket(k); err != nil {
					return
				}
				break // break "next loop" (= continue "seek loop")
			}
			if k, _ = c.Next(); k == nil {
				return // nothing more
			}

		}

	}

	return
}

func (d *driveFeeds) Roots(pk cipher.PubKey) UpdateRoots {
	r := new(driveRoots)
	r.feed = pk
	bk := d.bk.Bucket(pk[:])
	if bk == nil {
		return nil
	}
	r.bk = bk
	return r
}

type driveViewFeeds struct {
	*driveFeeds
}

func (d *driveViewFeeds) Roots(pk cipher.PubKey) ViewRoots {
	return d.driveFeeds.Roots(pk)
}

type driveRoots struct {
	feed cipher.PubKey
	bk   *bolt.Bucket
}

func (d *driveRoots) Feed() cipher.PubKey {
	return d.feed
}

func (d *driveRoots) Add(rp *RootPack) (err error) {

	// check

	if rp.Seq == 0 {
		if rp.Prev != (cipher.SHA256{}) {
			err = newRootError(d.feed, rp, "unexpected prev. reference")
			return
		}
	} else if rp.Prev == (cipher.SHA256{}) {
		err = newRootError(d.feed, rp, "missing prev. reference")
		return
	}
	hash := cipher.SumSHA256(rp.Root)
	if hash != rp.Hash {
		err = newRootError(d.feed, rp, "wrong hash of the root")
		return
	}
	data := encoder.Serialize(rp)
	seqb := utob(rp.Seq)

	// find

	if k, _ := d.bk.Cursor().Seek(seqb); bytes.Compare(k, seqb) != 0 {

		// not found

		err = d.bk.Put(seqb, data) // store
		return
	}

	// found (already exists)

	err = ErrRootAlreadyExists
	return
}

func (d *driveRoots) Last() (rp *RootPack) {
	_, last := d.bk.Cursor().Last()
	if last == nil {
		return // nil
	}
	rp = new(RootPack)
	if err := encoder.DeserializeRaw(last, rp); err != nil {
		panic(err) // critical
	}
	return
}

func (d *driveRoots) Get(seq uint64) (rp *RootPack) {
	seqb := utob(seq)
	_, data := d.bk.Cursor().Seek(seqb)
	if data == nil {
		return // nil
	}
	rp = new(RootPack)
	if err := encoder.DeserializeRaw(data, rp); err != nil {
		panic(err) // critical
	}
	return
}

func (d *driveRoots) Del(seq uint64) error {
	seqb := utob(seq)
	return d.bk.Delete(seqb)
}

func (d *driveRoots) Range(fn func(rp *RootPack) error) (err error) {

	var rp *RootPack
	c := d.bk.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(v, rp); err != nil {
			panic(err) // critical
		}
		if err = fn(rp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

func (d *driveRoots) Reverse(fn func(rp *RootPack) error) (err error) {

	var rp *RootPack
	c := d.bk.Cursor()
	for k, v := c.Last(); k != nil; k, v = c.Prev() {
		rp = new(RootPack)
		if err = encoder.DeserializeRaw(v, rp); err != nil {
			panic(err) // critical
		}
		if err = fn(rp); err != nil {
			if err == ErrStopRange {
				err = nil
			}
			return
		}
	}
	return
}

func (d *driveRoots) RangeDel(fn func(rp *RootPack) (bool, error)) (err error) {

	var rp *RootPack
	var del bool
	c := d.bk.Cursor()

	// seek loop
	for k, v := c.First(); k != nil; c.Seek(k) {

		// next loop
		for {

			rp = new(RootPack)
			if err = encoder.DeserializeRaw(v, rp); err != nil {
				panic(err) // critical
			}
			if del, err = fn(rp); err != nil {
				if err == ErrStopRange {
					err = nil
				}
				return
			}
			if del {
				if err = c.Delete(); err != nil {
					return
				}
				break // break "next loop" (= continue "seek loop")
			}
			if k, v = c.Next(); k == nil {
				return
			}

		}

	}

	return
}

func (d *driveRoots) DelBefore(seq uint64) (err error) {

	c := d.bk.Cursor()

	for k, _ := c.First(); k != nil; k, _ = c.Seek(k) {

		if btou(k) >= seq {
			return
		}

		if err = c.Delete(); err != nil {
			return
		}

	}

	return
}

/*

func (d *memoryDB) Del(key cipher.SHA256) {
	d.mx.Lock()
	defer d.mx.Unlock()

	delete(d.objects, key)
}

func (d *memoryDB) Get(key cipher.SHA256) (value []byte, ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	value, ok = d.objects[key] // unsafe
	return
}

func (d *memoryDB) Set(key cipher.SHA256, value []byte) {
	d.mx.Lock()
	defer d.mx.Unlock()

	d.objects[key] = value
}

func (d *memoryDB) Add(value []byte) (key cipher.SHA256) {
	key = cipher.SumSHA256(value)
	d.Set(key, value)
	return
}

func (d *memoryDB) IsExist(key cipher.SHA256) (ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	_, ok = d.objects[key]
	return
}

func (d *memoryDB) Range(fn func(key cipher.SHA256, value []byte) (stop bool)) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	for key, value := range d.objects {
		if fn(key, value) {
			return
		}
	}
	return
}

func (d *memoryDB) RangeDelete(fn func(key cipher.SHA256) (del bool)) {
	d.mx.Lock()
	defer d.mx.Unlock()

	for key := range d.objects {
		if fn(key) {
			delete(d.objects, key)
		}
	}
	return
}

//
// Feeds
//

func (d *memoryDB) AddFeed(pk cipher.PubKey) {
	d.mx.Lock()
	defer d.mx.Unlock()

	if _, ok := d.feeds[pk]; !ok {
		d.feeds[pk] = map[uint64]cipher.SHA256{}
	}
}

func (d *memoryDB) HasFeed(pk cipher.PubKey) (has bool) {
	d.mx.Lock()
	defer d.mx.Unlock()

	_, has = d.feeds[pk]
	return
}

func (d *memoryDB) Feeds() (fs []cipher.PubKey) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	if len(d.feeds) == 0 {
		return
	}
	fs = make([]cipher.PubKey, 0, len(d.feeds))
	for pk := range d.feeds {
		fs = append(fs, pk)
	}
	return
}

func (d *memoryDB) DelFeed(pk cipher.PubKey) {
	d.mx.Lock()
	defer d.mx.Unlock()

	for _, hash := range d.feeds[pk] {
		delete(d.roots, hash)
	}
	delete(d.feeds, pk)
}

func (d *memoryDB) AddRoot(pk cipher.PubKey, rr *RootPack) (err error) {
	rp := new(RootPack)
	*rp = *rr // copy (required)
	// test given rp
	if rp.Seq == 0 {
		if rp.Prev != (cipher.SHA256{}) {
			err = newRootError(pk, rp, "unexpected prev. reference")
			return
		}
	} else if rp.Prev == (cipher.SHA256{}) {
		err = newRootError(pk, rp, "missing prev. reference")
		return
	}
	if rp.Hash != cipher.SumSHA256(rp.Root) {
		err = newRootError(pk, rp, "wrong hash of the root")
		return
	}

	d.mx.Lock()
	defer d.mx.Unlock()

	//
	var ok bool
	var roots map[uint64]cipher.SHA256
	if roots, ok = d.feeds[pk]; !ok {
		d.feeds[pk] = map[uint64]cipher.SHA256{
			rp.Seq: rp.Hash,
		}
		d.roots[rp.Hash] = rp
		return
	}
	// ok
	if _, ok = roots[rp.Seq]; ok {
		err = ErrRootAlreadyExists
		return
	}
	roots[rp.Seq] = rp.Hash
	d.roots[rp.Hash] = rp
	return
}

func (d *memoryDB) LastRoot(pk cipher.PubKey) (rp *RootPack, ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	roots := d.feeds[pk]
	if len(roots) == 0 {
		return // bo roots
	}
	var max uint64
	for seq := range roots {
		if max < seq {
			max = seq
		}
	}
	if rp, ok = d.roots[roots[max]]; !ok {
		panic("broken db: misisng root") // critical
	}
	return
}

func (d *memoryDB) RangeFeed(pk cipher.PubKey,
	fn func(rp *RootPack) (stop bool)) {

	d.mx.RLock()
	defer d.mx.RUnlock()

	roots := d.feeds[pk]
	if len(roots) == 0 {
		return // empty feed
	}

	var o = make(order, 0, len(roots))
	for seq := range roots {
		o = append(o, seq)
	}
	sort.Sort(o)

	var rp *RootPack
	var ok bool

	for _, seq := range o {
		if rp, ok = d.roots[roots[seq]]; !ok {
			panic("broken database: misisng root") // critical
		}
		if fn(rp) {
			return // break
		}
	}
}

func (d *memoryDB) RangeFeedReverse(pk cipher.PubKey,
	fn func(rp *RootPack) (stop bool)) {

	d.mx.RLock()
	defer d.mx.RUnlock()

	roots := d.feeds[pk]
	if len(roots) == 0 {
		return // empty feed
	}

	var o = make(order, 0, len(roots))
	for seq := range roots {
		o = append(o, seq)
	}
	sort.Sort(sort.Reverse(o))

	var rp *RootPack
	var ok bool

	for _, seq := range o {
		if rp, ok = d.roots[roots[seq]]; !ok {
			panic("broken database: misisng root") // critical
		}
		if fn(rp) {
			return // break
		}
	}
}

func (d *memoryDB) RangeFeedDelete(pk cipher.PubKey,
	fn func(rp *RootPack) (del bool)) {

	d.mx.Lock()
	defer d.mx.Unlock()

	roots := d.feeds[pk]
	if len(roots) == 0 {
		return // empty feed
	}

	var o = make(order, 0, len(roots))
	for seq := range roots {
		o = append(o, seq)
	}
	sort.Sort(o)

	// ordered
	for _, seq := range o {
		rp := d.roots[roots[seq]]
		if fn(rp) {
			delete(d.roots, roots[seq])
			delete(roots, seq)
		}
	}

}

//
// Roots
//

func (d *memoryDB) GetRoot(hash cipher.SHA256) (rp *RootPack, ok bool) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	rp, ok = d.roots[hash]
	return
}

func (d *memoryDB) DelRootsBefore(pk cipher.PubKey, seq uint64) {
	d.mx.Lock()
	defer d.mx.Unlock()

	roots := d.feeds[pk]
	if len(roots) == 0 {
		return // empty feed
	}

	var o = make(order, 0, len(roots))
	for s := range roots {
		if s < seq {
			o = append(o, s)
		}
	}
	if len(o) == 0 {
		return
	}
	sort.Sort(o)

	for _, s := range o {
		delete(d.roots, roots[s])
		delete(roots, s)
	}
}

func (d *memoryDB) Stat() (s Stat) {
	d.mx.RLock()
	defer d.mx.RUnlock()

	s.Objects = len(d.objects)
	for _, v := range d.objects {
		s.Space += Space(len(v))
	}

	if len(d.feeds) == 0 {
		return
	}
	s.Feeds = make(map[cipher.PubKey]FeedStat, len(d.feeds))
	// lengths of Prev, Hash, Sig and Seq (8 byte)
	add := len(cipher.SHA256{})*2 + len(cipher.Sig{}) + 8
	for pk, rs := range d.feeds {
		var fs FeedStat
		for _, hash := range rs {
			fs.Space += Space(len(d.roots[hash].Root) + add)
		}
		fs.Roots = len(rs)
		s.Feeds[pk] = fs
	}
	return
}

func (d *memoryDB) Close() (_ error) {
	return
}

type order []uint64

func (o order) Len() int           { return len(o) }
func (o order) Less(i, j int) bool { return o[i] < o[j] }
func (o order) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }

*/
