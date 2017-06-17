package log

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

// defaults
const (
	Prefix string = ""    // default prefix
	Debug  bool   = false // don't show debug logs by default
)

// A Config represents configurationf for logger
type Config struct {
	Prefix string    // log prefix
	Debug  bool      // show debug logs
	Output io.Writer // provide an output
}

// NewConfig returns Config with defautl values
func NewConfig() (c Config) {
	c.Prefix = Prefix
	c.Debug = Debug
	return
}

// FromFlags parses commandline flags. So, you need to call
// flag.Parse after this method
func (c *Config) FromFlags() {
	flag.StringVar(&c.Prefix,
		"log-prefix",
		Prefix,
		"provide log-prefix")
	flag.BoolVar(&c.Debug,
		"debug",
		Debug,
		"print debug logs")
}

// A Logger is similar to log.Logger with Debug(ln|f)? methods
type Logger interface {
	// SetDebug is used to change debug logs flag. The method is not
	// safe for async usage
	SetDebug(bool)
	// IsDebug returns true if the logger in debug mode.
	// The method is not safe for async usage if you're
	// using SetDebug
	IsDebug() bool

	SetPrefix(string)    //
	SetFlags(int)        //
	SetOutput(io.Writer) //

	Print(...interface{})          //
	Println(...interface{})        //
	Printf(string, ...interface{}) //

	Panic(...interface{})          //
	Panicln(...interface{})        //
	Panicf(string, ...interface{}) //

	Fatal(...interface{})          //
	Fatalln(...interface{})        //
	Fatalf(string, ...interface{}) //

	Debug(...interface{})          //
	Debugln(...interface{})        //
	Debugf(string, ...interface{}) //
}

type logger struct {
	*log.Logger
	debug bool
}

// NewLogger create new Logger with given prefix and debug-enabling value.
// By default flags of the Logger is log.Lshortfile|log.Ltime
func NewLogger(prefix string, debug bool) Logger {
	return &logger{
		Logger: log.New(os.Stderr, prefix, log.Lshortfile|log.Ltime),
		debug:  debug,
	}
}

// NewLoggerByConfig create logger usinng given Config
func NewLoggerByConfig(conf Config) Logger {
	var out io.Writer = os.Stderr
	if conf.Output != nil {
		out = conf.Output
	}
	return &logger{
		Logger: log.New(out, conf.Prefix, log.Lshortfile|log.Ltime),
		debug:  conf.Debug,
	}
}

func (l *logger) Debug(args ...interface{}) {
	if l.debug {
		args = append([]interface{}{"[DBG] "}, args...)
		l.Output(2, fmt.Sprint(args...))
	}
}

func (l *logger) Debugln(args ...interface{}) {
	if l.debug {
		args = append([]interface{}{"[DBG]"}, args...)
		l.Output(2, fmt.Sprintln(args...))
	}
}

func (l *logger) Debugf(format string, args ...interface{}) {
	if l.debug {
		format = "[DBG] " + format
		l.Output(2, fmt.Sprintf(format, args...))
	}
}

func (l *logger) SetDebug(debug bool) {
	l.debug = debug
}

func (l *logger) IsDebug() bool { return l.debug }
