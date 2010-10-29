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
var loggerWriter io.Writer

func SetLogger(w io.Writer) {
	loggerWriter = w
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