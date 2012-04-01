all: pooch

clean:
	rm pooch

GOFILES=\
	dbg.go types.go dbname.go compat.go\
	parsetime.go tokenizer.go pureparser.go parserint.go\
	luaint.go backend.go\
	staticserve.go htmlformat.go serve.go multiserve.go\
	pooch.go

staticservedeps = static/* static/dot-luv/* static/dot-luv/images/*
staticserve.go: $(staticservedeps)
	perl make-staticserve.pl $(staticservedeps) > staticserve.go

pooch: $(GOFILES)
	go build && mv pooch2 pooch

