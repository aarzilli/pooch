/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */


package main

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
	return log.New(w, "", log.Ldate + log.Ltime)
}

var logger *log.Logger = makeLogger(os.Stderr)

func SetLogger(w io.Writer) {
	logger = makeLogger(w)
}

var CurrentLogLevel LogLevel = INFO

func Log(ll LogLevel, a ...interface{}) {
	if ll >= CurrentLogLevel {
		logger.Print(a...)
	}
}

func Logf(ll LogLevel, fmt string, a ...interface{}) {
	if ll >= CurrentLogLevel {
		logger.Printf(fmt, a...)
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


func CheckArgs(args []string, min int, max int, cmd string) {
	if min > -1 {
		if len(args) < min {
			Complain(false, "Not enough arguments for " + cmd + "\n")
		}
	}

	if max > -1 {
		if len(args) > max {
			Complain(false, "Too many arguments for " + cmd + "\n")
		}
	}
}

func Complain(usage bool, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	if usage {
		flag.Usage()
	}
	os.Exit(-1)
}

