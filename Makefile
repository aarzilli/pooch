all: pooch2

clean:
	rm pooch2

GOFILES=\
	pooch/dbg.go pooch/types.go pooch/dbname.go pooch/compat.go\
	pooch/parsetime.go pooch/tokenizer.go pooch/pureparser.go pooch/parserint.go\
	pooch/luaint.go pooch/backend.go\
	pooch/staticserve.go pooch/htmlformat.go pooch/serve.go pooch/multiserve.go\
	pooch/nfront.go pooch/ontology.go\
	pooch.go

staticservedeps = static/* static/dot-luv/* static/dot-luv/images/* static/jstree_default/*
pooch/staticserve.go: $(staticservedeps)
	perl make-staticserve.pl $(staticservedeps) > pooch/staticserve.go

pooch2: $(GOFILES)
	go build -o pooch2

