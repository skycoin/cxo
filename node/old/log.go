package node

import (
	"fmt"
	"io"
	"log"
	"os"
)

// A Logger is similar to log.Logger with Debug(ln|f)? methods;
// There are no mutexes for 'debug' flag
type Logger interface {
	SetDebug(bool)
	SetPrefix(string)
	SetFlags(int)
	SetOutput(io.Writer)

	Print(...interface{})
	Println(...interface{})
	Printf(string, ...interface{})

	Panic(...interface{})
	Panicln(...interface{})
	Panicf(string, ...interface{})

	Fatal(...interface{})
	Fatalln(...interface{})
	Fatalf(string, ...interface{})

	Debug(...interface{})
	Debugln(...interface{})
	Debugf(string, ...interface{})
}

type logger struct {
	*log.Logger
	debug bool
}

func NewLogger(prefix string, debug bool) Logger {
	return &logger{
		Logger: log.New(os.Stderr, prefix, log.Lshortfile|log.Ltime),
		debug:  debug,
	}
}

func (l *logger) Debug(args ...interface{}) {
	if l.debug {
		l.Output(2, fmt.Sprint(args...))
	}
}

func (l *logger) Debugln(args ...interface{}) {
	if l.debug {
		l.Output(2, fmt.Sprintln(args...))
	}
}

func (l *logger) Debugf(format string, args ...interface{}) {
	if l.debug {
		l.Output(2, fmt.Sprintf(format, args...))
	}
}

func (l *logger) SetDebug(debug bool) {
	l.debug = debug
}
