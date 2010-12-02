include $(GOROOT)/src/Make.inc

TARG=gotinyscheme

CGOFILES=tinyscheme.go

CGO_CFLAGS=-I../tinyscheme/
CGO_LDFLAGS=../tinyscheme/libtinyscheme.a -lm -ldl

include $(GOROOT)/src/Make.pkg