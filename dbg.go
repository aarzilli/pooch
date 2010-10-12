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
)

type LogLevel int

const (
	TRACE = LogLevel(iota)
	DEBUG
	INFO
	WARN
	ERROR
)
		
var CurrentLogLevel LogLevel = DEBUG
var LogDefault func(v...interface{}) = log.Stdout
var LogDefaultf func(fmt string, v...interface{}) = log.Stdoutf

func Log(ll LogLevel, a ...interface{}) {
	if ll >= CurrentLogLevel {
		LogDefault(a...)
	}
}

func Logf(ll LogLevel, fmt string, a ...interface{}) {
	if ll >= CurrentLogLevel {
		LogDefaultf(fmt, a...)
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