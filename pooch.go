package main

import (
	"flag"
	"fmt"
	"os"
	"tabwriter"
	"bufio"
	"io/ioutil"
	"strings"
	"container/vector"
	"path"
	"strconv"
)


var commands map[string](func (args []string) int) = map[string](func (args []string) int){
	"help": CmdHelp,
	"create": CmdCreate,
	"add": CmdAdd,
	"update": CmdUpdate,
	"get": CmdGet,
	"list": CmdList,
	"remove": CmdRemove,
	"import": CmdImport,
	"serve": CmdServe,
	"qadd": CmdQuickAdd,
	"search": CmdSearch,
	"tsvup": CmdTsvUpdate,
}

var help_commands map[string](func ()) = map[string](func ()){
	"help": HelpHelp,
	"create": HelpCreate,
	"add": HelpAdd,
	"update": HelpUpdate,
	"get": HelpGet,
	"list": HelpList,
	"remove": HelpRemove,
	"import": HelpImport,
	"serve": HelpServe,
	"qadd": HelpQuickAdd,
	"search": HelpSearch,
	"tsvup": HelpTsvUpdate,
}

func Complain(usage bool, format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, format, a)
	if usage {
		flag.Usage()
	}
	os.Exit(-1)
}

func CheckCondition(cond bool, format string, a ...interface{}) {
	if cond {
		fmt.Fprintf(os.Stderr, format, a)
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

func CheckArgsOpenDb(args []string, min int, max int, cmd string) *Tasklist {
	CheckArgs(args, min, max, cmd)

	name, found := Resolve(args[0])
	CheckCondition(!found, "Could not find database %s\n", name)

	return Open(name)
}

func CheckId(tl *Tasklist, id string, cmd string) {
	exists := tl.Exists(id)
	CheckCondition(!exists, "Cannot %s, id doesn't exists: %s\n", cmd, id)
}

func CmdCreate(args []string) int {
	CheckArgs(args, 1, 1, "create")

	filename, found := Resolve(args[0])
	
	Log(DEBUG, "Resolved filename: ", filename)

	CheckCondition(found, "Database already exists at: %s\n", filename)

	Create(filename)
	
	return 0
}

func HelpCreate() {
	fmt.Fprintf(os.Stderr, "Usage: create <db>\n\n")
	fmt.Fprintf(os.Stderr, "\tCreates a new empty db named <db>\n")
}

func ParseFile(in *os.File, id string) *Entry {
	buf, ioerr := ioutil.ReadAll(in)
	CheckCondition(ioerr != nil, "Error reading: %s\n", ioerr)
	s := string(buf)

	entry, parse_errors := Parse(s)

	fmt.Fprintf(os.Stderr, "%s\n", strings.Join(*parse_errors, "\n"))

	entry.SetId(id)

	return entry
}

func CmdAdd(args []string) int {
	tl := CheckArgsOpenDb(args, 1, 2, "add")
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

	tl.Add(ParseFile(os.Stdin, id))

	return 0
}

func HelpAdd() {
	fmt.Fprintf(os.Stderr, "Usage: add <db> <id>?\n\n")
	fmt.Fprintf(os.Stderr, "\tReads a new entry from standard input and adds it to <db> with the id <id>.\nIf <id> is not provided a new one is generated randomly. If the specified <id> collides with an existing id the command fails.\n")
}

func CmdQuickAdd(args []string) int {
	tl := CheckArgsOpenDb(args, 1, 1000, "qadd")
	defer tl.Close()

	qaddstr := strings.Join(args[1:], " ")

	Logf(DEBUG, "qadding: [%s]", qaddstr)

	entry, parse_errors := QuickParse(qaddstr)
	
	fmt.Fprintf(os.Stderr, "%s\n", strings.Join(*parse_errors, "\n"))
	
	entry.SetId(tl.MakeRandomId())

	tl.Add(entry)

	return 0
}

func HelpQuickAdd() {
	fmt.Fprintf(os.Stderr, "Usage: qadd <db> <quickadd string>\n\n")
	fmt.Fprintf(os.Stderr, "\tInterprest the quickadd string and adds it to the db")
}

func CmdSearch(args []string) int {
	tl := CheckArgsOpenDb(args, 1, 1000, "qadd")
	defer tl.Close()

	searchstr := strings.Join(args[1:], " ")

	Logf(DEBUG, "searching: [%s]", searchstr)

	CmdListEx(tl.Search(searchstr))

	return 0
}

func HelpSearch() {
	fmt.Fprintf(os.Stderr, "Usage: search <db> <search string>\n\n")
	fmt.Fprintf(os.Stderr, "\tLike list but only returns matching entries")
}

func CmdUpdate(args []string) int {
	tl := CheckArgsOpenDb(args, 2, 2, "update")
	defer tl.Close()

	id := args[1]
	CheckId(tl, id, "update")

	entry := ParseFile(os.Stdin, id)
	tl.Remove(id)
	tl.Add(entry)

	return 0
}

func HelpUpdate() {
	fmt.Fprintf(os.Stderr, "Usage: update <db> <id>\n\n")
	fmt.Fprintf(os.Stderr, "\tReads a new entry from standard input and uses it to replace the entry associated with <id> in the <db>.\nIf the specified <id> does not exist the command fails.\n")
}

func CmdGet(args []string) int {
	tl := CheckArgsOpenDb(args, 2, 2, "get")
	defer tl.Close()

	id := args[1]
	CheckId(tl, id, "get")

	entry := tl.Get(id)
	
	s := Deparse(entry)

	fmt.Printf("%s", s)

	return 0
}

func HelpGet() {
	fmt.Fprintf(os.Stderr, "Usage: get <db> <id>\n\n")
	fmt.Fprintf(os.Stderr, "\tPrints the entry associated with <id> inside <db>\n")
}

func CmdRemove(args []string) int {
	tl := CheckArgsOpenDb(args, 2, 2, "remove")
	defer tl.Close()

	id := args[1]
	CheckId(tl, id, "remove")

	tl.Remove(id)

	return 0
}

func HelpRemove() {
	fmt.Fprintf(os.Stderr, "Usage: remove <db> <id>\n\n")
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

func CmdList(argv []string) int {
	tl := CheckArgsOpenDb(argv, 1, 1, "list")
	defer tl.Close()

	CmdListEx(tl.GetList(false))

	return 0
}

func HelpList() {
	fmt.Fprintf(os.Stderr, "usage: list <db>\n\n")
	fmt.Fprintf(os.Stderr, "\tLists contents of <db>\n")
}

func CmdImport(argv []string) int {
	tl := CheckArgsOpenDb(argv, 2, 2, "import")
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

		tl.Add(ParseFile(file, info.Name))
	}
	
	return 0
}

func HelpImport() {
	fmt.Fprintf(os.Stderr, "usage: import <db> <directory>\n\n")
	fmt.Fprintf(os.Stderr, "\tImports the content of <directory> inside <db>\n\n")
}

func CmdServe(args []string) int {
	CheckArgs(args, 1, 20, "serve")

	port, converr := strconv.Atoi(args[0])
	CheckCondition(converr != nil, "Invalid port number %s: %s\n", args[0], converr)
	
	dbs := args[1:]
	names := make([]string, len(dbs))

	if len(dbs) > 0 {
		for i, db := range dbs {
			var found bool
			if dbs[i], found = Resolve(db); !found {
				fmt.Fprintf(os.Stderr, "Failed to resolve %s, excluded", db)
				dbs[i] = ""
			} else {
				names[i] = Base(dbs[i])
				fmt.Printf("Inserted %s as %s\n", dbs[i], names[i])
			}
		}
	} else {
		dbs, names = GetAllDefaultDBs()
	}

	fmt.Printf("Serve on: %d, %s\n", port, dbs)

	Serve(strconv.Itoa(port), names, dbs)

	return 0
}

func HelpServe() {
	fmt.Fprintf(os.Stderr, "usage: serve <port> <db>+\n\n")
	fmt.Fprintf(os.Stderr, "\tStarts http server for pooch on <port> with specified <db>s\n\n")
}

func CmdTsvUpdate(argv []string) int {
	tl := CheckArgsOpenDb(argv, 1, 1, "tsvup")
	defer tl.Close()

	in := bufio.NewReader(os.Stdin)

	for line, err := in.ReadString('\n'); err == nil; line, err = in.ReadString('\n') {
		entry := ParseTsvFormat(strings.Trim(line, "\t\n "));
		if tl.Exists(entry.Id()) {
			fmt.Printf("UPDATING\t%s\t%s\n", entry.Id(), entry.TriggerAt().Format("2006-01-02"))
			tl.Update(entry)
		} else {
			fmt.Printf("ADDING\t%s\t%s\n", entry.Id(), entry.TriggerAt().Format("2006-01-02"))
			tl.Add(entry)
		}
	}

	return 0
}

func HelpTsvUpdate() {
	fmt.Fprintf(os.Stderr, "usage: tsvup <db>\n\n")
	fmt.Fprintf(os.Stderr, "\tReads a tsv file from standard input and adds or updates every entry as specified\n")
	fmt.Fprintf(os.Stderr, "Entries should be in the form:\n")
	fmt.Fprintf(os.Stderr, "<id> <title> <priority> <at/sort>\n")
	fmt.Fprintf(os.Stderr, "The last field is interpreted as either a triggerAt field if priority is timed, or as the sort field otherwise\n\n")
}

func CmdHelp(args []string) int {
	CheckArgs(args, 0, 1, "help")
	if len(args) <= 0 {
		flag.Usage()
		return 0
	} else {
		helpfn := help_commands[args[0]]
		CheckCondition(helpfn == nil, "Unknown command: %s", args[0])
		helpfn()
	}
	return 0
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
		w.WriteString("\tcreate\tCreates new tasklist\n")
		w.WriteString("\tadd\tAdds entry to tasklist\n")
		w.WriteString("\tupdate\tUpdates entry in tasklist\n")
		w.WriteString("\tget\tDisplays an entry of the tasklist\n")
		w.WriteString("\tlist\tLists entries\n")
		w.WriteString("\tremove\tRemove entry\n")
		w.WriteString("\timprot\tImports directory in old pooch format\n")
		w.WriteString("\tserve\tStart http server\n")
		w.WriteString("\tqadd\tQuick add command\n")
		w.WriteString("\tsearch\tSearch\n")
		w.WriteString("\ttsvup\tAdd or update from tsv file\n")
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
