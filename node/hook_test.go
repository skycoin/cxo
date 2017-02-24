package node

import (
	"testing"
	"time"
)

type testHook struct {
	pend    chan struct{} // new pending conn
	estInc  chan struct{} // new establ inc
	estOut  chan struct{} // new establ out
	recvInc chan struct{} // receive inc
	recvOut chan struct{} // recv out
}

func newTestHook() (t *testHook) {
	t = new(testHook)
	t.pend = make(chan struct{}, 10)
	t.estInc = make(chan struct{}, 10)
	t.estOut = make(chan struct{}, 10)
	t.recvInc = make(chan struct{}, 10)
	t.recvOut = make(chan struct{}, 10)
	return
}

func (t *testHook) Hook(n *node, i int) {
	switch i {
	case testNewPending:
		t.pend <- struct{}{}
	case testOutgoingEstablished:
		t.estOut <- struct{}{}
	case testIncomingEstablished:
		t.estInc <- struct{}{}
	case testReceivedIncoming:
		t.recvInc <- struct{}{}
	case testReceivedOutgoign:
		t.recvOut <- struct{}{}
	}
}

func (t *testHook) chanTick(c <-chan struct{}, tk <-chan time.Time) bool {
	select {
	case <-c:
		return true
	case <-tk:
	}
	return false
}

func (t *testHook) chanTimeout(c chan struct{}, n int, tm time.Duration) bool {
	tk := time.NewTicker(tm)
	defer tk.Stop()
	for n > 0 {
		if !t.chanTick(c, tk.C) {
			return false
		}
		n--
	}
	return true
}

func (h *testHook) pendTimeout(n int, tm time.Duration,
	t *testing.T) (ok bool) {

	if ok = h.chanTimeout(h.pend, n, tm); !ok {
		t.Errorf("haven't got %d pend conns after %v", n, tm)
	}
	return
}

func (h *testHook) estIncTimeout(n int, tm time.Duration,
	t *testing.T) (ok bool) {

	if ok = h.chanTimeout(h.estInc, n, tm); !ok {
		t.Errorf("haven't got %d est incoming conns after %v", n, tm)
	}
	return
}

func (h *testHook) estOutTimeout(n int, tm time.Duration,
	t *testing.T) (ok bool) {

	if ok = h.chanTimeout(h.estOut, n, tm); !ok {
		t.Errorf("haven't got %d est outging conns after %v", n, tm)
	}
	return
}

func (h *testHook) recvIncTimeout(n int, tm time.Duration,
	t *testing.T) (ok bool) {

	if ok = h.chanTimeout(h.recvInc, n, tm); !ok {
		t.Errorf("haven't got %d received msgs on incoming conn after %v",
			n, tm)
	}
	return
}

func (h *testHook) recvOutTimeout(n int, tm time.Duration,
	t *testing.T) (ok bool) {

	if ok = h.chanTimeout(h.recvOut, n, tm); !ok {
		t.Errorf("haven't got %d received msgs on outging conn after %v", n, tm)
	}
	return
}

func (t *testHook) Set(n Node) {
	n.(*node).testHook = t.Hook
}
