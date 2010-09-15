/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */


package main

import (
	"log"
)

type LogLevel int

const (
	TRACE = LogLevel(iota)
	DEBUG
	INFO
	WARN
	ERROR
)
		
var CurrentLogLevel LogLevel = TRACE
var LogDefault func(v...interface{}) = log.Stdout
var LogDefaultf func(fmt string, v...interface{}) = log.Stdoutf

func Log(ll LogLevel, a ...interface{}) {
	if ll >= CurrentLogLevel {
		LogDefault(a)
	}
}

func Logf(ll LogLevel, fmt string, a ...interface{}) {
	if ll >= CurrentLogLevel {
		LogDefaultf(fmt, a)
	}
}

