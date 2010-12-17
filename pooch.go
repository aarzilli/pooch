/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"flag"
	"fmt"
	"os"
	"tabwriter"
	"bufio"
	"strings"
	"strconv"
	"json"
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
	"colist": CmdColist,
	"tsvup": CmdTsvUpdate,
	"rename": CmdRename,
	"rentag": CmdRenTag,
	
	"multiserve": CmdMultiServe,
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
	"colist": HelpColist,
	"tsvup": HelpTsvUpdate,
	"rename": HelpRename,
	"rentag": HelpRenTag,
	"compat": CompatHelp,
	"multiserve": HelpMultiServe,
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
		entry, parse_errors := QuickParse(strings.Join(args[0:], " "), "", nil, 0)
		
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(*parse_errors, "\n"))
		
		entry.SetId(tl.MakeRandomId())
		
		tl.Add(entry)
	})
}

func HelpQuickAdd() {
	fmt.Fprintf(os.Stderr, "Usage: add <quickadd string>\n\n")
	fmt.Fprintf(os.Stderr, "\tInterprets the quickadd string and adds it to the db")
}

func CmdQuickUpdate(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 1, 1000, "update", func (tl *Tasklist, args []string, flags map[string]bool) {
		CheckId(tl, args[0], "update")
		
		entry, parse_errors := QuickParse(strings.Join(args[1:], " "), "", nil, 0)
		
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(*parse_errors, "\n"))
		
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
		timezone := tl.GetTimezone()
		includeDone := flags["d"]; tsv := flags["t"]; js := flags["j"]

		showCols := make(map[string]bool)
		
		theselect, query := SearchParse(strings.Join(args[0:], " "), includeDone, false, nil, showCols, tl)
		
		Logf(DEBUG, "Search statement [%s] with query [%s]\n", theselect, query)

		entries := tl.Retrieve(theselect, query)
		switch {
		case tsv: CmdListExTsv(entries, showCols, timezone)
		case js: CmdListExJS(entries, timezone)
		default: CmdListEx(entries, showCols, timezone)
		}
	})
}

func HelpSearch() {
	fmt.Fprintf(os.Stderr, "Usage: search [-dtj] <search string>\n\n")
	fmt.Fprintf(os.Stderr, "\tReturns a list of matching entries\n")
	fmt.Fprintf(os.Stderr, "\t-t\tWrites output in tsv format\n")
	fmt.Fprintf(os.Stderr, "\t-d\tIncludes done entries\n")
	fmt.Fprintf(os.Stderr, "\t-j\tPrints JSON\n")
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

func CmdColist(args []string) {
	CheckArgsOpenDb(args, map[string]bool{}, 0, 1, "colist", func(tl *Tasklist, args []string, flags map[string]bool) {
		theselect, base := "", ""
		set := make(map[string]string)
		if len(args) > 0 {
			base = args[0]
			_, theselect = SearchParseToken("+"+base, tl, set, make(map[string]bool), false)
		}
		
		subcols := tl.GetSubcols(theselect);
		
		for _, x := range subcols {
			if _, ok := set[x]; ok { continue }
			fmt.Printf("%s#%s\n", base, x)
		}
	})
}

func HelpColist() {
	fmt.Fprintf(os.Stderr, "Usage: colist <list of columns>\n\n")
	fmt.Fprintf(os.Stderr, "\tReturns categories associated with the list of columns (the list of columns colud be empty)")
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

func GetSizesForList(v []*Entry, showCols map[string]bool) (id_size, title_size, cat_size int, colSizes map[string]int) {
	title_size, id_size, cat_size = 0, 0, 0
	colSizes = make(map[string]int)
	for showCol, _ := range showCols {
		colSizes[showCol] = len(showCol)
	}
	for _, e := range v {
		if len(e.Title())+1 > title_size { title_size = len(e.Title())+1 }
		if len(e.Id())+1 > id_size { id_size = len(e.Id())+1 }
		if len(e.CatString())+1 > cat_size { cat_size = len(e.CatString())+1 }

		for showCol, _ := range showCols {
			if len(e.Columns()[showCol])+1 > colSizes[showCol] { colSizes[showCol] = len(e.Columns()[showCol])+1 }
		}
	}

	return
}

var LINE_SIZE int = 80

func CmdListExTsv(v []*Entry, showCols map[string]bool, timezone int) {
	fmt.Printf("\t\t")
	for showCol, _ := range showCols {
		fmt.Printf("\t%s", showCol)
	}
	fmt.Printf("\n")
	
	for _, entry := range v {
		timeString := TimeString(entry.TriggerAt(), entry.Sort(), timezone)
		fmt.Printf("%s\t%s\t%s", entry.Id(), entry.Title(), timeString)
		for showCol, _ := range showCols {
			fmt.Printf("\t%s", entry.Columns()[showCol])
		}
		fmt.Printf("\n")
	}
}

func CmdListEx(v []*Entry, showCols map[string]bool, timezone int) {
	id_size, title_size, cat_size, col_sizes := GetSizesForList(v, showCols)

	var curp Priority = INVALID

	fmt.Printf("%s %s %s %s", RepeatString(" ", id_size), RepeatString(" ", title_size), RepeatString(" ", 19), RepeatString(" ", cat_size))
	for showCol, colSize := range col_sizes {
		fmt.Printf(" %s%s", showCol, RepeatString(" ", colSize - len(showCol)))
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

		for showCol, colSize := range col_sizes {
			fmt.Printf(" %s%s", entry.Columns()[showCol], RepeatString(" ", colSize - len(entry.Columns()[showCol])))
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
	logfile, err := os.Open(args[2], os.O_WRONLY|os.O_CREAT|os.O_APPEND, 0666)
	CheckCondition(err != nil, "Couldn't open logfile %s: %s\n", logfile, err)
	defer logfile.Close()
	
	/*
	logfileBuffered := bufio.NewWriter(logfile)
	defer logfileBuffered.Flush()

	SetLogger(logfileBuffered)*/
	SetLogger(logfile)
	
	MultiServe(args[0], args[1])
}

func HelpMultiServe() {
	fmt.Fprintf(os.Stderr, "usage: multiserve <port> <directory> <logfile>\n\n")
	fmt.Fprintf(os.Stderr, "\tStarts a multi-user http server, information will be stored in <directory>. Writes logs to <logfile>\n\n")
}

func CmdTsvUpdate(argv []string) {
	CheckArgsOpenDb(argv, map[string]bool{}, 0, 0, "tsvup", func (tl *Tasklist, args []string, flags map[string]bool) {
		in := bufio.NewReader(os.Stdin)
		
		for line, err := in.ReadString('\n'); err == nil; line, err = in.ReadString('\n') {
			entry := ParseTsvFormat(strings.Trim(line, "\t\n "), tl, tl.GetTimezone());
			if tl.Exists(entry.Id()) {
				fmt.Printf("UPDATING\t%s\t%s\n", entry.Id(), entry.TriggerAt().Format("2006-01-02"))
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
		w.WriteString("\tserve\tStart http server\n")
		w.WriteString("\tmultiserve\tStart multiuser http server\n")

		w.Flush()
		tw.Flush()
	}

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
			WriteStackTrace(rerr, loggerWriter)
			os.Exit(-1)
		}
	}()
	
	fn(args[1:])
}
