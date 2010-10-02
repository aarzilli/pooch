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
	"container/vector"
	"strconv"
)

var commands map[string](func (args []string)) = map[string](func (args []string)){
	"help": CmdHelp,
	"create": CmdCreate,
	"get": CmdGet,
	"remove": CmdRemove,
	"serve": CmdServe,
	"add": CmdQuickAdd,
	"update": CmdQuickUpdate,
	"search": CmdSearch,
	"colist": CmdColist,
	"tsvup": CmdTsvUpdate,
	"rename": CmdRename,

	"multiserve": CmdMultiServe,

	"port": CmdPort,
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
	"colist": HelpColist,
	"tsvup": HelpTsvUpdate,
	"rename": HelpRename,
	"compat": CompatHelp,

	"port": HelpPort,
}

func Complain(usage bool, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a...)
	if usage {
		flag.Usage()
	}
	os.Exit(-1)
}

func CheckCondition(cond bool, format string, a ...interface{}) {
	if cond {
		fmt.Fprintf(os.Stderr, format, a...)
		os.Exit(-1)
	}
}

func CheckArgs(args []string, min int, max int, cmd string) {
	if min > -1 {
		if len(args) < min {
			Complain(true, "Not enough arguments for " + cmd + "\n")
		}
	}

	if max > -1 {
		if len(args) > max {
			Complain(true, "Too many arguments for " + cmd + "\n")
		}
	}
}

func CheckArgsOpenDb(args []string, min int, max int, cmd string, rest func(tl *Tasklist)) {
	CheckArgs(args, min, max, cmd)
	WithOpenDefault(rest)
}

func CheckId(tl *Tasklist, id string, cmd string) {
	exists := tl.Exists(id)
	CheckCondition(!exists, "Cannot %s, id doesn't exists: %s\n", cmd, id)
}

func CmdCreate(args []string) {
	CheckArgs(args, 1, 1, "create")

	filename, found := Resolve(args[0])
	
	Log(DEBUG, "Resolved filename: ", filename)

	CheckCondition(found, "Database already exists at: %s\n", filename)

	Create(filename)
}

func HelpCreate() {
	fmt.Fprintf(os.Stderr, "Usage: create <db>\n\n")
	fmt.Fprintf(os.Stderr, "\tCreates a new empty db named <db>\n")
}

func CmdQuickAdd(args []string) {
	CheckArgsOpenDb(args, 0, 1000, "add", func(tl *Tasklist) {
		entry, parse_errors := QuickParse(strings.Join(args[0:], " "), "", nil)
		
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
	CheckArgsOpenDb(args, 1, 1000, "update", func (tl *Tasklist) {
		CheckId(tl, args[0], "update")
		
		entry, parse_errors := QuickParse(strings.Join(args[1:], " "), "", nil)
		
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(*parse_errors, "\n"))
		
		entry.SetId(args[0])
		tl.Update(entry)
	})
}

func HelpQuickUpdate() {
	fmt.Fprintf(os.Stderr, "Usage: update <id> <quickadd string>\n\n")
	fmt.Fprintf(os.Stderr, "\tInterprets the quickadd string and updates selected entry in the db")
}

func CmdSearch(args []string) {
	CheckArgsOpenDb(args, 0, 1000, "search", func(tl *Tasklist) {
		theselect, query := SearchParse(strings.Join(args[0:], " "), false, false, nil, tl)

		Logf(DEBUG, "Search statement [%s] with query [%s]\n", theselect, query)

		CmdListEx(tl.Retrieve(theselect, query))
	})
}

func HelpSearch() {
	fmt.Fprintf(os.Stderr, "Usage: search <search string>\n\n")
	fmt.Fprintf(os.Stderr, "\tReturns a list of matching entries")
}

func CmdColist(args []string) {
	CheckArgsOpenDb(args, 0, 1, "colist", func(tl *Tasklist) {
		theselect, base := "", ""
		set := make(map[string]string)
		if len(args) > 0 {
			base = args[0]
			_, theselect = SearchParseToken("+"+base, tl, set, false)
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
	CheckArgsOpenDb(args, 1, 1, "remove", func (tl *Tasklist) {
		CheckId(tl, args[0], "remove")
		tl.Remove(args[0])
	})
}

func HelpRemove() {
	fmt.Fprintf(os.Stderr, "Usage: remove <id>\n\n")
	fmt.Fprintf(os.Stderr, "\tRemoves specified entry from <db>\n")
}

func GetSizesForList(v *vector.Vector) (id_size int, title_size int) {
	title_size, id_size = 0, 0
	for _, entry := range *v {
		var e *Entry;
		e = entry.(*Entry)
		if len(e.Title()) > title_size {
			title_size = len(e.Title())
		}

		if len(e.Id()) > id_size {
			id_size = len(e.Id())
		}
	}

	return
}

var LINE_SIZE int = 80

func CmdListEx(v *vector.Vector) {
	id_size, title_size := GetSizesForList(v)

	spare_size := LINE_SIZE - id_size - title_size - len(TRIGGER_AT_SHORT_FORMAT)

	if (title_size != 0) && (id_size != 0) {
		if spare_size > 0 {
			title_size += spare_size * title_size / (title_size + id_size)
			id_size += spare_size * id_size / (title_size + id_size)
		}
	}

	var curp Priority = INVALID
	
	for _, e := range *v {
		entry := e.(*Entry)

		if entry.Priority() != curp {
			curp = entry.Priority()
			fmt.Printf("\n%s:\n", strings.ToUpper(curp.String()))
		}

		
		timeString := TimeString(entry.TriggerAt(), entry.Sort())
		
		fmt.Printf("%s%s %s%s %s\n",
			entry.Id(), strings.Repeat(" ", id_size - len(entry.Id())),
			entry.Title(), strings.Repeat(" ", title_size - len(entry.Title())),
			timeString)
	}
}

func CmdServe(args []string) {
	CheckArgs(args, 1, 1, "serve")

	port, converr := strconv.Atoi(args[0])
	CheckCondition(converr != nil, "Invalid port number %s: %s\n", args[0], converr)
	Serve(strconv.Itoa(port))
}

func HelpServe() {
	fmt.Fprintf(os.Stderr, "usage: serve <port>\n\n")
	fmt.Fprintf(os.Stderr, "\tStarts http server for pooch on <port>\n\n")
}

func CmdMultiServe(args []string) {
	//TODO: check things right
	MultiServe(args[0], args[1])
}

func HelpMultiServe() {
	fmt.Fprintf(os.Stderr, "usage: multiserve <port> <directory>\n\n")
	fmt.Fprintf(os.Stderr, "\tStarts a multi-user http server, information will be stored in <directory>\n\n")
}

func CmdTsvUpdate(argv []string) {
	CheckArgsOpenDb(argv, 0, 0, "tsvup", func (tl *Tasklist) {
		in := bufio.NewReader(os.Stdin)
		
		for line, err := in.ReadString('\n'); err == nil; line, err = in.ReadString('\n') {
			entry := ParseTsvFormat(strings.Trim(line, "\t\n "), tl);
			if tl.Exists(entry.Id()) {
				fmt.Printf("UPDATING\t%s\t%s\n", entry.Id(), entry.TriggerAt().Format("2006-01-02"))
				tl.Update(entry)
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
	CheckArgsOpenDb(argv, 1, 2, "rename", func (tl *Tasklist) {
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

func CmdGet(args []string) {
	CheckArgsOpenDb(args, 1, 1, "get", func(tl *Tasklist) {
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
	CheckArgs(args, 0, 1, "help")
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
		w.WriteString("\tcolist\tReturns lists of categories\n")
		w.WriteString("\n")
		w.WriteString("\tget\tDisplays an entry of the tasklist\n")
		w.WriteString("\tadd\tAdd command\n")
		w.WriteString("\tupdate\tUpdate command\n")
		w.WriteString("\ttsvup\tAdd or update from tsv file\n")
		w.WriteString("\tremove\tRemove entry\n")
		w.WriteString("\n")
		w.WriteString("\tserve\tStart http server\n")

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
			os.Exit(-1)
		}
	}()
	
	fn(args[1:])
}
