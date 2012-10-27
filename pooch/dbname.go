/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package pooch

import (
	"os"
	"path"
	"strings"
)

func FixExtension(name string) string {
	if (len(name) >= 6) && (name[len(name)-6:] == ".pooch") {
		return name
	}

	return name + ".pooch"
}

func WithOpenDefault(rest func(tl *Tasklist)) {
	dbname := os.Getenv("POOCHDB")
	if dbname == "" { panic("POOCHDB Not Set") }
	tl := OpenOrCreate(dbname)
	defer tl.Close()
	tl.mutex.Lock()
	defer tl.mutex.Unlock()
	rest(tl)
}

func SearchFile(name string) (outname string, found bool) {
	outname = name
	found = false

	poochpaths := strings.Split(os.Getenv("POOCHPATH"), ":")
	for _, curpath := range poochpaths {
		attempt := path.Join(curpath, outname)
		_, err := os.Stat(attempt)
		if err == nil {
			outname = attempt
			found = true
		}
	}

	return
}

func Resolve(name string) (outname string, found bool) {
	found = false
	outname = FixExtension(name)
	outname, found = SearchFile(outname)
	return
}

func Base(dbpath string) string {
	name := path.Base(dbpath)
	return name[0:(len(name)-len(".pooch"))]
}


