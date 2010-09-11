include $(GOROOT)/src/Make.inc

TARG=pooch
#TARG=parsemain

GOFILES=\
	dbg.go types.go parse.go\
	dbname.go backend.go staticserve.go htmlformat.go serve.go compat.go\
	pooch.go

include $(GOROOT)/src/Make.cmd

staticservedeps = static-test.html list.css dlist.css int.js shortcut.js json.js calendar.css calendar.js fullcalendar.css fullcalendar.js jquery.js jquery-ui-custom.js cint.js cal.css
staticserve.go: $(staticservedeps)
	perl make-staticserve.pl $(staticservedeps) > staticserve.go

