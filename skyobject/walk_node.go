package skyobject

import (
	"errors"
	"fmt"
)

type unsaver interface {
	unsave() (err error)
}

type walkNode struct {
	value interface{} // golang value or Value (skyobject.Value)
	upper unsaver     // upepr node (for Refs trees)
	sch   Schema      // schema of the value
	pack  *Pack       // back reference to related Pack
}

func (w *walkNode) unsave() (err error) {
	if up := w.upper; up != nil {
		err = up.unsave()
	}
	return
}

// Ref, Refs or Dynamic in updateStack
type commiter interface {
	commit() (err error)
}

// track changes
type updateStack struct {
	stack    []commiter
	contains map[commiter]struct{}
}

func (u *updateStack) init() {
	u.contains = make(map[commiter]struct{})
}

func (u *updateStack) Push(ref interface{}) (err error) {
	switch tt := ref.(type) {
	case *Ref:
		if tt.wn == nil {
			return errors.New("Push uninitialized Ref")
		}
	case *Refs:
		if tt.wn == nil {
			return errors.New("Push uninitialized Refs")
		}
	case *Dynamic:
		if tt.wn == nil {
			return errors.New("Push uninitialized Dynamic")
		}
	default:
		return fmt.Errorf("push invalid type of reference %T", ref)
	}
	cm, ok := ref.(commiter)
	if !ok {
		panic(fmt.Errorf("%T does not implements commiter interface", ref))
	}
	if _, ok := u.contains[cm]; ok {
		return // already have
	}
	u.stack = append(u.stack, cm)
	u.contains[cm] = struct{}{}
	return
}

// Push pointer to Ref, Refs or Dynamic to track changes
func (p *Pack) Push(ref interface{}) (err error) {
	if p.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}
	return p.updateStack.Push(ref)
}

func (u updateStack) Pop() (last interface{}) {
	if len(u.stack) == 0 {
		return
	}
	last = u.stack[len(u.stack)-1]
	u.stack[len(u.stack)-1] = nil // for golagn GC
	u.stack = u.stack[len(u.stack)-1:]
	delete(u.contains, last.(commiter))
	return
}

// Pop last element to undo last Push (e.g. don't
// track changes for last element pushed)
func (p *Pack) Pop() (last interface{}) {
	if p.flags&ViewOnly != 0 {
		return
	}
	return p.updateStack.Pop()
}

func (u *updateStack) Commit() (err error) {
	for i := len(u.stack) - 1; i >= 0; i-- {
		if err = u.stack[i].commit(); err != nil {
			u.stack = u.stack[:i]
		}
		delete(u.contains, u.stack[i])
		u.stack[i] = nil // golang GC
	}
	u.stack = u.stack[:0]
	return
}

// Commit all unsaved chagnes and clear the stack.
// It stops on first error. If there is an error then
// the stack will be cleared before the erroneous
// reference
func (p *Pack) Commit() (err error) {
	if p.flags&ViewOnly != 0 {
		return ErrViewOnlyTree
	}
	return p.updateStack.Commit()
}

func (u *updateStack) ClearStack() {
	for i := range u.stack {
		u.stack[i] = nil
	}
	u.stack = u.stack[:0]
	u.contains = make(map[commiter]struct{})
}

// ClearStack clears the stack
func (p *Pack) ClearStack() {
	if p.flags&ViewOnly != 0 {
		return
	}
	p.updateStack.ClearStack()
}
