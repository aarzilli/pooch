/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"fmt"
	"os"
	"path"
	"container/vector"
	"strings"
	"time"
	"io/ioutil"
	"tabwriter"
	"bufio"
)

func FileList(pathToDir string) []os.FileInfo {
	dir, operr := os.Open(pathToDir, os.O_RDONLY, 0666)

	if operr != nil {
		panic(fmt.Sprintf("Error opening directory %s: %s", pathToDir, operr))
	}
	
	defer dir.Close()

	infos, rderr := dir.Readdir(-1)

	if rderr != nil {
		panic(fmt.Sprintf("Error reding directory %s: %s", pathToDir, rderr))
	}

	return infos
}

func CompatParseFile(in *os.File, id string) *Entry {
	buf, ioerr := ioutil.ReadAll(in)
	CheckCondition(ioerr != nil, "Error reading: %s\n", ioerr)
	s := string(buf)

	entry, parse_errors := CompatParse(s)

	fmt.Fprintf(os.Stderr, "%s\n", strings.Join(*parse_errors, "\n"))

	entry.SetId(id)

	return entry
}

func CmdAdd(args []string) int {
	tl := CheckArgsOpenDb(args, 1, 2, "old-add")
	defer tl.Close()
	
	var id string
	if len(args) > 1 {
		id = args[1]
		exists := tl.Exists(id)
		CheckCondition(exists, "Cannot add, id already exists: %s\n", id)
	} else {
		id = tl.MakeRandomId()
	}

	Log(DEBUG, "Adding:", id)

	tl.Add(CompatParseFile(os.Stdin, id))

	return 0
}

func HelpAdd() {
	fmt.Fprintf(os.Stderr, "Usage: old-add <db> <id>?\n\n")
	fmt.Fprintf(os.Stderr, "\tReads a new entry from standard input and adds it to <db> with the id <id>.\nIf <id> is not provided a new one is generated randomly. If the specified <id> collides with an existing id the command fails.\n")
}

func CmdUpdate(args []string) int {
	tl := CheckArgsOpenDb(args, 2, 2, "old-update")
	defer tl.Close()

	id := args[1]
	CheckId(tl, id, "update")

	entry := CompatParseFile(os.Stdin, id)
	tl.Update(entry)

	return 0
}

func HelpUpdate() {
	fmt.Fprintf(os.Stderr, "Usage: old-update <db> <id>\n\n")
	fmt.Fprintf(os.Stderr, "\tReads a new entry from standard input and uses it to replace the entry associated with <id> in the <db>.\nIf the specified <id> does not exist the command fails.\n")
}

func CmdImport(argv []string) int {
	tl := CheckArgsOpenDb(argv, 2, 2, "old-import")
	defer tl.Close()

	dirname := argv[1]

	infos := FileList(dirname)

	for _, info := range infos {
		if info.IsDirectory() { continue }

		filename := path.Join(dirname, info.Name)

		exists := tl.Exists(info.Name)
		if exists { continue }

		file, operr := os.Open(filename, os.O_RDONLY, 0666)
		if operr != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Can not open %s for import\n", filename)
			continue
		}
		defer file.Close()

		tl.Add(CompatParseFile(file, info.Name))
	}
	
	return 0
}

func HelpImport() {
	fmt.Fprintf(os.Stderr, "usage: old-import <db> <directory>\n\n")
	fmt.Fprintf(os.Stderr, "\tImports the content of <directory> inside <db>\n\n")
}

func CmdPort(args []string) int {
	CheckArgs(args, 2, 2, "create")
	
	filename, _ := Resolve(args[0])
	
	Log(DEBUG, "Resolved filename: ", filename)

	Port(filename, args[1])
	
	return 0

}

func HelpPort() {
	fmt.Fprintf(os.Stderr, "porting older dbs\n")
}

func CmdCompatGet(args []string) int {
	tl := CheckArgsOpenDb(args, 2, 2, "get")
	defer tl.Close()

	id := args[1]
	CheckId(tl, id, "get")

	entry := tl.Get(id)
	
	fmt.Printf("%s", CompatDeparse(entry))

	return 0
}

func HelpCompatGet() {
	fmt.Fprintf(os.Stderr, "Usage: old-get <db> <id>\n\n")
	fmt.Fprintf(os.Stderr, "\tPrints the entry associated with <id> inside <db>\n")
}


func CompatParse(input string) (*Entry, *vector.StringVector) {
	lines := strings.Split(input, "\n", -1)

	title := ""
	text := ""
	priority := NOW
	var freq Frequency = 0
	var sort string = ""
	var triggerAt *time.Time = nil
	errors := new(vector.StringVector)
	
	for lineno, cur := range lines {
		lineno++
		switch true {
		case (cur != "") && (title == ""):
			title = cur
			Log(DEBUG, "Title found at line: ", lineno)
		case (len(cur) > 0) && (cur[0] == '&'):
			Log(DEBUG, "Command found at line: ", lineno)
			ss := strings.Split(cur[1:], " ", 2)
			directive, argument := ss[0], ss[1]
			switch directive {
			case "sort":
				sort = argument
			case "at":
				var err string
				triggerAt, err = ParseDateTime(argument)
				if (err != "") {
					errors.Push(fmt.Sprintf("line %d: %s", lineno, err))
				}
			case "repeat", "recur":
				var err string
				freq, err = ParseFrequency(argument)
				if (err != "") {
					errors.Push(fmt.Sprintf("line %d: %s", lineno, err))
				}
			case "priority":
				var err string
				priority, err = ParsePriority(argument)
				if (err != "") {
					errors.Push(fmt.Sprintf("line %d: %s", lineno, err))
				}
			default:
				Log(DEBUG, "Unknown directive: [", directive, "]")
				errors.Push(fmt.Sprintf("line %d: Unknown directive: %s", lineno, directive))
			}
		default:
			if (text != "") || (cur != "") {
				text += cur + "\n"
			}
		}
	}

	if title == "" {
		title = "(nil)"
	}

	text = strings.TrimSpace(text)

	if sort == "" {
		sort = SortFromTriggerAt(triggerAt)
	}


	if (priority == INVALID) {
		priority = NOW
	}

	Log(DEBUG, "Title:     ", title)
	Log(DEBUG, "Priority:  ", priority.String())
	if triggerAt != nil {
		Log(DEBUG, "At:        ", triggerAt.String())
	}
	Log(DEBUG, "Frequency: ", freq.String())
	Log(DEBUG, "Sort:      ", sort)
	Log(DEBUG, "Text:      ", text)
	Log(DEBUG, "Errors:    ", errors)

	return MakeEntry("", title, text, priority, freq, triggerAt, sort, make(Columns)), errors
}


func CompatDeparse(entry *Entry) string {
	r := fmt.Sprintln(entry.Title())
	
	if entry.Text() != "" {
		r += fmt.Sprintln("")
		r += fmt.Sprintln(entry.Text())
	}
	
	r += fmt.Sprintln("")
	
	priority := entry.Priority()
	r += fmt.Sprintln("&priority", priority.String())
	
	freq := entry.Freq()
	r += fmt.Sprintln("&freq", freq.ToInteger())
	
	triggerAt := entry.TriggerAt()
	if (triggerAt != nil) {
		r += fmt.Sprintln("&at", triggerAt.Format(TRIGGER_AT_FORMAT))
	}
	
	r += fmt.Sprintln("&sort", entry.Sort())
	return r
}

func CompatHelp() {
	tw := tabwriter.NewWriter(os.Stderr, 8, 8, 4, ' ', 0)
	w := bufio.NewWriter(tw)

	w.WriteString("\told-add\tAdds entry to tasklist\n")
	w.WriteString("\told-update\tUpdates entry in tasklist\n")
	w.WriteString("\told-improt\tImports directory in old pooch format\n")
	w.WriteString("\told-get\tPrints entry (format compatible with old-add)\n")
	w.WriteString("\tport\tAdds a columns table to a database that doesn't have it and populates it with base informations\n")

	w.Flush()
	tw.Flush()
}
