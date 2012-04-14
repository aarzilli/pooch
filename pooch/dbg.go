/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package pooch

import (
	"log"
	"io"
	"fmt"
	"runtime"
	"os"
	"flag"
)

type LogLevel int

const (
	TRACE = LogLevel(iota)
	DEBUG
	INFO
	WARN
	ERROR
)

func makeLogger(w io.Writer) *log.Logger {
	return log.New(w, "", log.Ldate + log.Ltime + log.Lmicroseconds)
}

var Logger *log.Logger = makeLogger(os.Stderr)
var LoggerWriter io.Writer

func SetLogger(w io.Writer) {
	LoggerWriter = w
	Logger = makeLogger(w)
}

var CurrentLogLevel LogLevel = INFO

func Log(ll LogLevel, a ...interface{}) {
	if ll >= CurrentLogLevel {
		Logger.Print(a...)
	}
}

func Logf(ll LogLevel, fmt string, a ...interface{}) {
	if ll >= CurrentLogLevel {
		Logger.Printf(fmt, a...)
	}
}

func WriteStackTrace(rerr interface{}, out io.Writer) {
	fmt.Fprintf(out, "Stack trace for: %s\n", rerr)
	for i := 1; ; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok { break }
		fmt.Fprintf(out, "    %s:%d\n", file, line)
	}
}


func CheckArgs(args []string, accepted map[string]bool, min int, max int, cmd string) (nargs []string, flags map[string]bool) {
	nargs = []string{}
	flags = make(map[string]bool)

	for _, arg := range args {
		if (arg[0] == '-') && (len(arg) > 1) {
			arg = arg[1:len(arg)]
			if v, ok := accepted[arg]; v && ok {
				flags[arg] = true
			} else {
				Complain(false, "Unknown flag " + arg + "\n")
			}
		} else {
			nargs = append(nargs, arg)
		}
	}

	if min > -1 {
		if len(nargs) < min {
			Complain(false, "Not enough arguments for " + cmd + "\n")
		}
	}

	if max > -1 {
		if len(nargs) > max {
			Complain(false, "Too many arguments for " + cmd + "\n")
		}
	}

	return
}

func Complain(usage bool, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	if usage {
		flag.Usage()
	}
	os.Exit(-1)
}

