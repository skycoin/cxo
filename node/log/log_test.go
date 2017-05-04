package log

import (
	"bytes"
	"fmt"
	"testing"
)

func cleanLoggerOut(prefix string, debug bool) (l Logger, out *bytes.Buffer) {
	out = new(bytes.Buffer)
	l = NewLogger(prefix, debug)
	l.SetOutput(out)
	l.SetFlags(0)
	return
}

func TestNewLogger(t *testing.T) {
	var l Logger
	t.Run("prefix", func(t *testing.T) {
		var prefix string = "prefix"
		var out *bytes.Buffer
		l, out = cleanLoggerOut(prefix, false)
		l.Print()
		if out.String() != fmt.Sprintln("prefix") { // add line break
			t.Errorf("wrong prefix: want %q, got %q", prefix, out.String())
		}
	})
	t.Run("debug", func(t *testing.T) {
		l = NewLogger("", false)
		if d := l.(*logger).debug; d != false {
			t.Errorf("wrong debug flag: want false, got %q", d)
		}
		l = NewLogger("", true)
		if d := l.(*logger).debug; d != true {
			t.Errorf("wrong debug flag: want true, got %q", d)
		}
	})
}

func TestLogger_Debug(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		l, out := cleanLoggerOut("", true)
		l.Debug("some")
		if some := "[DBG] " + fmt.Sprintln("some"); some != out.String() {
			t.Errorf("want %q, got %q", some, out.String())
		}
	})
	t.Run("false", func(t *testing.T) {
		l, out := cleanLoggerOut("", false)
		l.Debug("some")
		if out.String() != "" {
			t.Error("got debug message even it disabled")
		}
	})
}

func TestLogger_Debugln(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		l, out := cleanLoggerOut("", true)
		l.Debugln("some")
		if some := "[DBG] " + fmt.Sprintln("some"); some != out.String() {
			t.Errorf("want %q, got %q", some, out.String())
		}
	})
	t.Run("false", func(t *testing.T) {
		l, out := cleanLoggerOut("", false)
		l.Debugln("some")
		if out.String() != "" {
			t.Error("got debug message even it disabled")
		}
	})
}

func TestLogger_Debugf(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		l, out := cleanLoggerOut("", true)
		l.Debugf("some")
		if some := "[DBG] " + fmt.Sprintln("some"); some != out.String() {
			t.Errorf("want %q, got %q", some, out.String())
		}
	})
	t.Run("false", func(t *testing.T) {
		l, out := cleanLoggerOut("", false)
		l.Debugf("some")
		if out.String() != "" {
			t.Error("got debug message even it disabled")
		}
	})
}

func TestLogger_SetDebug(t *testing.T) {
	var l Logger = NewLogger("", false)
	l.SetDebug(true)
	if d := l.(*logger).debug; d != true {
		t.Errorf("wrong debug flag: want true, got %q", d)
	}
	l.SetDebug(false)
	if d := l.(*logger).debug; d != false {
		t.Errorf("wrong debug flag: want false, got %q", d)
	}
}
