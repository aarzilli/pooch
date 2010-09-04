package main

import (
	"os"
	"path"
	"strings"
	"container/vector"
)

func FixExtension(name string) string {
	if (len(name) >= 6) && (name[len(name)-6:] == ".pooch") {
		return name
	}
	
	return name + ".pooch"
}

func SearchFile(name string) (outname string, found bool) {
	outname = name
	found = false

	poochpaths := strings.Split(os.Getenv("POOCHPATH"), ":", -1)
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

func GetAllDefaultDBs() (dbs []string, names []string) {
	vdbs := (*vector.StringVector)(&dbs)
	vnames := (*vector.StringVector)(&names)
	
	for _, curpath := range strings.Split(os.Getenv("POOCHPATH"), ":", -1) {
		curdir, err := os.Open(curpath, os.O_RDONLY, 0)
		if err != nil { continue }

		infos, rderr := curdir.Readdir(-1)
		if rderr != nil { continue }

		for _, info := range infos {
			matches, _ := path.Match("*.pooch", info.Name)
			if !matches { continue }

			dbpath := path.Join(curpath, info.Name)
			Log(DEBUG, "adding:", dbpath)
			vdbs.Push(dbpath)
			vnames.Push(Base(dbpath))
		}
	}

	return dbs, names
}

