/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"bufio"
	"strings"
	"strconv"
	"encoding/json"
	"io/ioutil"
	. "./pooch"
	//"runtime"
)

//import _ "http/pprof"

var commands map[string](func (args []string)) = map[string](func (args []string)){
	"help": CmdHelp,
	"create": CmdCreate,
	"get": CmdGet,
	"remove": CmdRemove,
	"serve": CmdServe,
	"add": CmdQuickAdd,
	"update": CmdQuickUpdate,
	"search": CmdSearch,
	"savesearch": CmdSaveSearch,
	"tsvup": CmdTsvUpdate,
	"rename": CmdRename,
	"rentag": CmdRenTag,
	"errlog": CmdErrorLog,

	"multiserve": CmdMultiServe,
	"multiserveplain": CmdMultiServePlain,

	"setopt": CmdSetOption,
	"getopt": CmdGetOption,

	"run": CmdRun,
}

var help_commands map[string](func ()) = map[string](func ()){
	"help": HelpHelp,
	"create": HelpCreate,
	"get": HelpGet,
	"remove": HelpRemove,
	"serve": HelpServe,
	"add": HelpQuickAdd,
	"update": HelpQuickUpdate,
	"search": HelpSearch,
	"savesearch": HelpSaveSearch,
	"tsvup": HelpTsvUpdate,
	"rename": HelpRename,
	"rentag": HelpRenTag,
	"errlog": HelpErrorLog,
	"compat": CompatHelp,
	"multiserve": HelpMultiServe,
	"multiserveplain": HelpMultiServePlain,
	"setopt": HelpSetOption,
	"getopt": HelpGetOption,
	"run": HelpRun,
}

func CheckCondition(cond bool, format string, a ...interface{}) {
	if cond {
		fmt.Fprintf(os.Stderr, format, a...)
		os.Exit(-1)
	}
}

func CheckArgsOpenDb(args []string, flags map[string]bool, min int, max int, cmd string, rest func(tl *Tasklist, args []string, flags map[string]bool)) {
	nargs, nflags := CheckArgs(args, flags, min, max, cmd)
	WithOpenDefault(func(tl *Tasklist) { rest(tl, nargs, nflags) })
}

func CheckId(tl *Tasklist, id string, cmd string) {
	exists := tl.Exists(id)
	CheckCondition(!exists, "Cannot %s, id doesn't exists: %s\n", cmd, id)
}

func CmdCreate(args []string) {
	CheckArgs(args, map[string]bool{}, 1, 1, "create")

	filename, found := Resolve(args[0])

	Log(DEBUG, "Resolved filename: ", filename)

	CheckCondition(found, "Database already exists at: %s\n", filename)

	tasklist := OpenOrCreate(filename)
	tasklist.Close()
}

func HelpCreate() {
	fmt.Fprintf(os.Stderr, "Usage: create <db>\n\n")
	fmt.Fprintf(os.Stderr, "\tCreates a new empty db named <db>\n")
}

func CmdQuickAdd(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 0, 1000, "add", func(tl *Tasklist, args []string, flags map[string]bool) {
		var entry *Entry
		if (len(args) == 1) && (args[0] == "-") {
			entry = tl.ExtendedAddParse()
		} else {
			entry  = tl.ParseNew(strings.Join(args[0:], " "), "")
		}
		tl.Add(entry)
		Logf(INFO, "Added entry: %s\n", entry.Id())
	})
}

func HelpQuickAdd() {
	fmt.Fprintf(os.Stderr, "Usage: add <quickadd string>\n\n")
	fmt.Fprintf(os.Stderr, "\tInterprets the quickadd string and adds it to the db. Using a single - as the quickadd string makes the program read the quickadd string from stdin, in a special format that allows easier setting of columns\n")
}

func CmdQuickUpdate(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 1, 1000, "update", func (tl *Tasklist, args []string, flags map[string]bool) {
		CheckId(tl, args[0], "update")

		entry := tl.ParseNew(strings.Join(args[1:], " "), "")

		entry.SetId(args[0])
		tl.Update(entry, false)
	})
}

func HelpQuickUpdate() {
	fmt.Fprintf(os.Stderr, "Usage: update <id> <quickadd string>\n\n")
	fmt.Fprintf(os.Stderr, "\tInterprets the quickadd string and updates selected entry in the db")
}

func CmdSearch(args []string) {
	CheckArgsOpenDb(args, map[string]bool{ "t": true, "d": true, "j": true }, 0, 1000, "search", func(tl *Tasklist, args []string, flags map[string]bool) {
		var input string
		if (len(args) == 1) && (args[0] == "-") {
			buf, err := ioutil.ReadAll(os.Stdin)
			Must(err)
			input = string(buf)
		} else {
			input = strings.Join(args[0:], " ")
		}

		timezone := tl.GetTimezone()
		tsv := flags["t"]; js := flags["j"]

		theselect, command, _, _, _, showCols, _, perr := tl.ParseSearch(input, nil)
		Must(perr)

		Logf(DEBUG, "Search statement\n%s\n", theselect)

		entries, serr := tl.Retrieve(theselect, command)
		Must(serr)

		switch {
		case tsv: CmdListExTsv(entries, showCols, timezone)
		case js: CmdListExJS(entries, timezone)
		default: CmdListEx(entries, showCols, timezone)
		}
	})
}

func HelpSearch() {
	fmt.Fprintf(os.Stderr, "Usage: search [-tj] <search string>\n\n")
	fmt.Fprintf(os.Stderr, "\tReturns a list of matching entries\n")
	fmt.Fprintf(os.Stderr, "\t-t\tWrites output in tsv format\n")
	fmt.Fprintf(os.Stderr, "\t-j\tPrints JSON\n")
	fmt.Fprintf(os.Stderr, "Using a single - as the search string will make the program read the search string from standard input")
}

func CmdSaveSearch(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 1, 1000, "savesearch", func(tl *Tasklist, args []string, flags map[string]bool) {
		tl.SaveSearch(args[0], strings.Join(args[1:], " "))
	})
}

func HelpSaveSearch() {
	fmt.Fprintf(os.Stderr, "Usage: savesearch <name> <search string>\n\n")
	fmt.Fprintf(os.Stderr, "\tSaves the search string as <name>\n")
}

func CmdRemove(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 1, 1, "remove", func (tl *Tasklist, args []string, flags map[string]bool) {
		CheckId(tl, args[0], "remove")
		tl.Remove(args[0])
	})
}

func HelpRemove() {
	fmt.Fprintf(os.Stderr, "Usage: remove <id>\n\n")
	fmt.Fprintf(os.Stderr, "\tRemoves specified entry from <db>\n")
}

func GetSizesForList(v []*Entry, showCols []string) (id_size, title_size, cat_size int, colSizes map[string]int) {
	title_size, id_size, cat_size = 0, 0, 0
	colSizes = make(map[string]int)
	for _, showCol := range showCols {
		colSizes[showCol] = len(showCol)
	}
	for _, e := range v {
		if len(e.Title())+1 > title_size { title_size = len(e.Title())+1 }
		if len(e.Id())+1 > id_size { id_size = len(e.Id())+1 }
		if len(e.CatString())+1 > cat_size { cat_size = len(e.CatString())+1 }

		for _, showCol := range showCols {
			if len(e.Columns()[showCol])+1 > colSizes[showCol] { colSizes[showCol] = len(e.Columns()[showCol])+1 }
		}
	}

	return
}

var LINE_SIZE int = 80

func CmdListExTsv(v []*Entry, showCols []string, timezone int) {
	fmt.Printf("\t\t")
	for showCol, _ := range showCols {
		fmt.Printf("\t%s", showCol)
	}
	fmt.Printf("\n")

	for _, entry := range v {
		timeString := TimeString(entry.TriggerAt(), entry.Sort(), timezone)
		fmt.Printf("%s\t%s\t%s", entry.Id(), entry.Title(), timeString)
		for _, showCol := range showCols {
			fmt.Printf("\t%s", entry.Columns()[showCol])
		}
		fmt.Printf("\n")
	}
}

func CmdListEx(v []*Entry, showCols []string, timezone int) {
	id_size, title_size, cat_size, col_sizes := GetSizesForList(v, showCols)

	var curp Priority = INVALID

	fmt.Printf("%s %s %s %s", RepeatString(" ", id_size), RepeatString(" ", title_size), RepeatString(" ", 19), RepeatString(" ", cat_size))
	for _, colName := range showCols {
		fmt.Printf(" %s%s", colName, RepeatString(" ", col_sizes[colName] - len(colName)))
	}
	fmt.Printf("\n")

	for _, entry := range v {
		if entry.Priority() != curp {
			curp = entry.Priority()
			fmt.Printf("\n%s:\n", strings.ToUpper(curp.String()))
		}

		timeString := TimeString(entry.TriggerAt(), entry.Sort(), timezone)

		fmt.Printf("%s%s %s%s %s%s %s%s",
			entry.Id(), RepeatString(" ", id_size - len(entry.Id())),
			entry.Title(), RepeatString(" ", title_size - len(entry.Title())),
			timeString, RepeatString(" ", 19 - len(timeString)),
			entry.CatString(), RepeatString(" ", cat_size - len(entry.CatString())))

		for _, colName := range showCols {
			fmt.Printf(" %s%s", entry.Columns()[colName], RepeatString(" ", col_sizes[colName] - len(entry.Columns()[colName])))
		}

		fmt.Printf("\n")
	}
}

func CmdListExJS(v []*Entry, timezone int) {
	for _, entry := range v {
		json.NewEncoder(os.Stdout).Encode(MarshalEntry(entry, timezone))
	}
}

func CmdServe(args []string) {
	CheckArgs(args, map[string]bool{}, 1, 1, "serve")

	port, converr := strconv.Atoi(args[0])
	CheckCondition(converr != nil, "Invalid port number %s: %s\n", args[0], converr)
	Serve(strconv.Itoa(port))
}

func HelpServe() {
	fmt.Fprintf(os.Stderr, "usage: serve <port>\n\n")
	fmt.Fprintf(os.Stderr, "\tStarts http server for pooch on <port>\n\n")
}

func CmdMultiServe(args []string) {
	CheckArgs(args, map[string]bool{}, 3, 3, "multiserve")
	_, converr := strconv.Atoi(args[0])
	CheckCondition(converr != nil, "Invalid port number %s: %s\n", args[0], converr)
	logfile, err := os.OpenFile(args[2], os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	CheckCondition(err != nil, "Couldn't open logfile %s: %s\n", logfile, err)
	defer logfile.Close()

	SetLogger(logfile)

	MultiServe(args[0], args[1])
}


func HelpMultiServe() {
	fmt.Fprintf(os.Stderr, "usage: multiserve <port> <directory> <logfile>\n\n")
	fmt.Fprintf(os.Stderr, "\tStarts a multi-user http server, information will be stored in <directory>. Writes logs to <logfile>\n\n")
}

func CmdMultiServePlain(args []string) {
	SecureCookies = false
	CmdMultiServe(args)
}

func HelpMultiServePlain() {
	fmt.Fprintf(os.Stderr, "usage: multiserveplain <port> <directory> <logfile>\n\n")
	fmt.Fprintf(os.Stderr, "\tJust like multiserve, but cookies are stored insecurely (allows using multiserve without an https proxy)\n")
}

func CmdTsvUpdate(argv []string) {
	CheckArgsOpenDb(argv, map[string]bool{}, 0, 0, "tsvup", func (tl *Tasklist, args []string, flags map[string]bool) {
		in := bufio.NewReader(os.Stdin)

		for line, err := in.ReadString('\n'); err == nil; line, err = in.ReadString('\n') {
			line = strings.Trim(line, "\t\n ")
			if line == "" { continue }

			entry := ParseTsvFormat(line, tl, tl.GetTimezone());
			if tl.Exists(entry.Id()) {
				//fmt.Printf("UPDATING\t%s\t%s\n", entry.Id(), entry.TriggerAt().Format("2006-01-02"))
				tl.Update(entry, false)
			} else {
				fmt.Printf("ADDING\t%s\t%s\n", entry.Id(), entry.TriggerAt().Format("2006-01-02"))
				tl.Add(entry)
			}
		}
	})
}

func HelpTsvUpdate() {
	fmt.Fprintf(os.Stderr, "usage: tsvup\n\n")
	fmt.Fprintf(os.Stderr, "\tReads a tsv file from standard input and adds or updates every entry as specified\n")
	fmt.Fprintf(os.Stderr, "Entries should be in the form:\n")
	fmt.Fprintf(os.Stderr, "<id> <title> <priority> <at/sort>\n")
	fmt.Fprintf(os.Stderr, "The last field is interpreted as either a triggerAt field if priority is timed, or as the sort field otherwise\n\n")
}

func CmdRename(argv []string) {
	CheckArgsOpenDb(argv, map[string]bool{}, 1, 2, "rename", func (tl *Tasklist, args []string, flags map[string]bool) {
		src_id := argv[0]
		dst_id := tl.MakeRandomId()
		if len(argv) > 1 { dst_id = argv[1] }

		entry := tl.Get(src_id)
		tl.Remove(entry.Id())
		entry.SetId(dst_id)
		tl.Add(entry)
	})
}

func HelpRename() {
	fmt.Fprintf(os.Stderr, "usage : rename src_id [dst_id]\n")
	fmt.Fprintf(os.Stderr, "\tRenames src_id into dst_id (or a random id if nothing is specified\n")
}

func CmdRenTag(argv []string) {
	CheckArgsOpenDb(argv, map[string]bool{}, 2, 2, "rentag", func (tl *Tasklist, args []string, flags map[string]bool) {
		src_tag := argv[0]
		dst_tag := argv[1]
		tl.RenameTag(src_tag, dst_tag)
	})
}

func CmdErrorLog(argv []string) {
	CheckArgsOpenDb(argv, map[string]bool{}, 0, 0, "errlog", func (tl *Tasklist, args []string, flags map[string]bool) {
		errors := tl.RetrieveErrors()
		for _, error := range errors {
			fmt.Printf("%s\t%s\n", error.TimeString(), error.Message)
		}
	})
}

func HelpErrorLog() {
	fmt.Fprintf(os.Stderr, "usage: errlog\n")
	fmt.Fprintf(os.Stderr, "\tShows error log\n")
}

func HelpRenTag() {
	fmt.Fprintf(os.Stderr, "usage: rentag src_tag dst_tag\n")
	fmt.Fprintf(os.Stderr, "\tRenames <src_tag> to <dst_tag>\n")
}

func CmdGet(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 1, 1, "get", func(tl *Tasklist, args []string, flags map[string]bool) {
		id := args[0]
		CheckId(tl, id, "get")

		entry := tl.Get(id)
		entry.Print()
	})
}

func HelpGet() {
	fmt.Fprintf(os.Stderr, "Usage: get <id>\n\n")
	fmt.Fprintf(os.Stderr, "\tPrints the entry associated with <id> inside <db>\n")
}

func CmdSetOption(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 2, 2, "setopt", func(tl *Tasklist, args []string, flags map[string]bool) {
		name := args[0]
		value := args[1]

		private := false

		if strings.Index(name, "private:") == 0 {
			name = name[len("private:"):len(name)]
			private = true
		}

		if value == "-" {
			buf, err := ioutil.ReadAll(os.Stdin)
			Must(err)
			value = string(buf)
		}

		if private {
			tl.SetPrivateSetting(name, value)
		} else {
			tl.SetSetting(name, value)
		}
	})
}

func HelpSetOption() {
	fmt.Fprintf(os.Stderr, "Usage: setopt <name> <value>\n\n")
	fmt.Fprintf(os.Stderr, "\tSets <name> option to <value>. Prefix <name> with 'private:' if you want to change a private option\n")
	fmt.Fprintf(os.Stderr, "If <value> is '-' will read from standard input.\n")
}

func CmdGetOption(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 1, 1, "getopt", func(tl *Tasklist, args []string, flags map[string]bool) {
		name := args[0]

		private := false

		if strings.Index(name, "private:") == 0 {
			name = name[len("private:"):len(name)]
			private = true
		}

		if private {
			fmt.Printf("%s\n", tl.GetPrivateSetting(name))
		} else {
			fmt.Printf("%s\n", tl.GetSetting(name))
		}
	})
}


func HelpGetOption() {
	fmt.Fprintf(os.Stderr, "Usage: getopt <name>\n\n")
	fmt.Fprintf(os.Stderr, "\tReturns value of option <name>. Prefix <name> with 'private:' if you want to see a private option\n")
}

func CmdRun(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 1, 1000, "run", func(tl *Tasklist, args []string, flags map[string]bool) {
		fname := args[0]

		fentry := tl.Get(fname)
		tl.DoRunString(fentry.Text(), args[1:len(args)])

		if tl.ShowReturnValueRequest() {
			entries, cols := tl.LuaResultToEntries()
			CmdListEx(entries, cols, tl.GetTimezone())
		}
	})
}

func HelpRun() {
	fmt.Fprintf(os.Stderr, "Usage: run <function id> <args>\n\n")
	fmt.Fprintf(os.Stderr, "\tRuns function passing arguments\n")
}

func CmdHelp(args []string) {
	CheckArgs(args, map[string]bool{}, 0, 1, "help")
	if len(args) <= 0 {
		flag.Usage()
	} else {
		helpfn := help_commands[args[0]]
		CheckCondition(helpfn == nil, "Unknown command: %s", args[0])
		helpfn()
	}
}

func HelpHelp() {
	fmt.Fprintf(os.Stderr, "Usage: help <command>?\n\n")
	fmt.Fprintf(os.Stderr, "\tDisplays help about <command>, if the argument is omitted displays the list of commands, with brief descriptions of each\n")
}

func main() {
	SetLogger(os.Stderr)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		tw := tabwriter.NewWriter(os.Stderr, 8, 8, 4, ' ', 0)
		w := bufio.NewWriter(tw)
		w.WriteString("\thelp\tHelp system\n")
		w.WriteString("\thelp compat\tShows backwards compatibility commands\n")
		w.WriteString("\n")
		w.WriteString("\tcreate\tCreates new tasklist\n")
		w.WriteString("\n")
		w.WriteString("\tsearch\tSearch (lists everything when called without a query)\n")
		w.WriteString("\tsavesearch\tSaves a search string\n")
		w.WriteString("\tcolist\tReturns lists of categories\n")
		w.WriteString("\n")
		w.WriteString("\tget\tDisplays an entry of the tasklist\n")
		w.WriteString("\tadd\tAdd command\n")
		w.WriteString("\tupdate\tUpdate command\n")
		w.WriteString("\ttsvup\tAdd or update from tsv file\n")
		w.WriteString("\tremove\tRemove entry\n")
		w.WriteString("\trename\tRename entry\n")
		w.WriteString("\trentag\tRename tags\n")
		w.WriteString("\n")
		w.WriteString("\tsetopt\tSets options\n")
		w.WriteString("\n")
		w.WriteString("\tserve\tStart http server\n")
		w.WriteString("\tmultiserve\tStart multiuser http server\n")
		w.WriteString("\tmultiserveplain\tStart multiuser http server, does not request secure cookies\n")
		w.WriteString("\n")
		w.WriteString("\tsetopt\tSets option\n")
		w.WriteString("\tgetopt\tGets option value\n")
		w.WriteString("\n")
		w.WriteString("\trun\tRuns functions\n")

		w.Flush()
		tw.Flush()
	}

	//fmt.Printf("Processors: %d\n", runtime.GOMAXPROCS(0))

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(-1)
	}

	args := flag.Args()
	fn := commands[args[0]]
	CheckCondition(fn == nil, "Unknown command: %s\n", args[0])

	defer func() {
		if rerr := recover(); rerr != nil {
			fmt.Fprintf(os.Stderr, "Error executing command %s: %s\n", args[0], rerr)
			WriteStackTrace(rerr, LoggerWriter)
			os.Exit(-1)
		}
	}()

	fn(args[1:])
}
