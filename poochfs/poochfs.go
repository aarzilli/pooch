package main

import (
	"fmt"
	"os"
	"path/filepath"
	"rsc.io/rsc/fuse"
	"strings"
	"sync"
	"unicode"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: poochfs mountpoint\n")
		os.Exit(1)
	}
	if os.Getenv("POOCHSRV") == "" {
		fmt.Fprintf(os.Stderr, "POOCHSRV not set\n")
		os.Exit(1)
	}
	if os.Getenv("POOCHKEY") == "" {
		fmt.Fprintf(os.Stderr, "POOCHKEY not set\n")
		os.Exit(1)
	}
	c, err := fuse.Mount(os.Args[1])
	must(err)

	c.Serve(FS{})
}

type FS struct {
}

func (FS) Root() (fuse.Node, fuse.Error) {
	return Dir{"/"}, nil
}

var inodeMap = map[string]uint64{}
var nextInode = uint64(4)
var mu sync.Mutex

func convertToDirent(v []string, prefix string, typ uint32) []fuse.Dirent {
	mu.Lock()
	defer mu.Unlock()
	r := make([]fuse.Dirent, len(v))
	for i := range v {
		p := filepath.Join(prefix, v[i])
		inode, ok := inodeMap[p]
		if !ok {
			inodeMap[p] = nextInode
			inode = nextInode
			nextInode++
		}
		r[i] = fuse.Dirent{Inode: inode, Name: v[i], Type: typ}
	}
	return r
}

type Dir struct {
	path string
}

func (Dir) Attr() fuse.Attr {
	return fuse.Attr{Mode: os.ModeDir | 0555}
}

func (d Dir) Lookup(name string, intr fuse.Intr) (fuse.Node, fuse.Error) {
	switch d.path {
	case "/":
		switch name {
		case "tag":
			return Dir{"/tag"}, nil
		case "query":
			return Dir{"/query"}, nil
		}
	case "/tag":
		return QueryDir{"#" + name}, nil
	case "/query":
		return QueryDir{"#%" + name}, nil
	}
	return nil, fuse.ENOENT
}

func (d Dir) ReadDir(intr fuse.Intr) ([]fuse.Dirent, fuse.Error) {
	switch d.path {
	case "/":
		return []fuse.Dirent{
			{Inode: 2, Name: "tag", Type: 4},
			{Inode: 3, Name: "query", Type: 4},
		}, nil
	case "/tag":
		_, tags := readOntology()
		return convertToDirent(tags, "/tag", 4), nil
	case "/query":
		queries, _ := readOntology()
		return convertToDirent(queries, "/query", 4), nil
	}
	return nil, fuse.ENOENT
}

type QueryDir struct {
	query string
}

func (QueryDir) Attr() fuse.Attr {
	return fuse.Attr{Mode: os.ModeDir | 0555}
}

func (d QueryDir) Lookup(name string, intr fuse.Intr) (fuse.Node, fuse.Error) {
	v := strings.SplitN(name, "_", 3)
	if len(v) < 2 {
		return nil, fuse.ENOENT
	}
	return EntryFile{v[1]}, nil
}

func (d QueryDir) ReadDir(intr fuse.Intr) ([]fuse.Dirent, fuse.Error) {
	out := readList(d.query)
	entries := make([]fuse.Dirent, len(out.Results))
	for i, res := range out.Results {
		p := filepath.Join("/entry/", res.Id)
		inode, ok := inodeMap[p]
		if !ok {
			inodeMap[p] = nextInode
			inode = nextInode
			nextInode++
		}
		entries[i] = fuse.Dirent{
			Inode: inode,
			Type:  0,
			Name:  fmt.Sprintf("%s_%s_%s", res.Priority.String(), res.Id, fixtitle(res.Title)),
		}
	}
	return entries, nil
}

func fixtitle(title string) string {
	in := []rune(title)
	out := make([]rune, len(in))
	for i, ch := range in {
		if unicode.IsLetter(ch) || unicode.IsNumber(ch) {
			out[i] = ch
		} else {
			out[i] = '_'
		}
	}
	return string(out)
}

type EntryFile struct {
	id string
}

func (EntryFile) Attr() fuse.Attr {
	return fuse.Attr{Mode: 0444}
}

func (f EntryFile) ReadAll(intr fuse.Intr) ([]byte, fuse.Error) {
	out := readList("#:id=" + f.id)
	if len(out.Results) != 1 {
		return nil, fuse.EIO
	}
	return []byte(out.Results[0].Title + "\n\n" + out.Results[0].Text), nil
}

func (f EntryFile) WriteAll(buf []byte, intr fuse.Intr) fuse.Error {
	if saveEntry(f.id, string(buf)) != nil {
		return fuse.EIO
	}
	return nil
}
