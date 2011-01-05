include $(GOROOT)/src/Make.inc

TARG=pooch
#TARG=parsemain

GOFILES=\
	dbg.go types.go dbname.go compat.go\
	parsetime.go tokenizer.go pureparser.go parserint.go\
	luaint.go backend.go\
	staticserve.go htmlformat.go serve.go multiserve.go\
	pooch.go

include $(GOROOT)/src/Make.cmd

staticservedeps = static/*
staticserve.go: $(staticservedeps)
	perl make-staticserve.pl $(staticservedeps) > staticserve.go

