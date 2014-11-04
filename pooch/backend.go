/*
 This program is distributed under the terms of GPLv3
 Copyright 2010 - 2013, Alessandro Arzilli
*/

package pooch

import (
	"code.google.com/p/gosqlite/sqlite"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aarzilli/golua/lua"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Tasklist struct {
	filename              string
	conn                  *sqlite.Conn
	luaState              *lua.State
	luaFlags              *LuaFlags
	mutex                 *sync.Mutex
	refs                  int
	timestamp             int64
	executionLimitEnabled bool
	curCut                string
}

var enabledCaching bool = true
var tasklistCache map[string]*Tasklist = make(map[string]*Tasklist)
var tasklistCacheMutex sync.Mutex

func MustExec(conn *sqlite.Conn, stmt string, v ...interface{}) {
	Must(conn.Exec(stmt, v...))
}

func HasTable(conn *sqlite.Conn, name string) bool {
	stmt, err := conn.Prepare("SELECT name FROM sqlite_master WHERE name = ?")
	Must(err)
	defer stmt.Finalize()
	Must(stmt.Exec(name))
	return stmt.Next()
}

func (tasklist *Tasklist) MustExec(stmt string, v ...interface{}) {
	MustExec(tasklist.conn, stmt, v...)
}

func (tasklist *Tasklist) WithTransaction(f func()) {
	tasklist.MustExec("BEGIN EXCLUSIVE TRANSACTION")
	defer func() {
		if rerr := recover(); rerr != nil {
			Logf(ERROR, "Rolling back a failed transaction, because of %v\n", rerr)
			tasklist.conn.Exec("ROLLBACK TRANSACTION")
			panic(rerr)
		} else {
			Logf(DEBUG, "Transaction committed\n")
			tasklist.MustExec("COMMIT TRANSACTION")
		}
	}()

	f()
}

func internalTasklistOpenOrCreate(filename string) *Tasklist {
	conn, err := sqlite.Open(filename)
	Must(err)

	if !HasTable(conn, "errorlog") { // optimization, if the last added table exists exists do not try to create anything
		MustExec(conn, "CREATE TABLE IF NOT EXISTS tasks(id TEXT PRIMARY KEY, title_field TEXT, text_field TEXT, priority INTEGER, trigger_at_field DATE, sort TEXT);")
		MustExec(conn, "CREATE INDEX IF NOT EXISTS tasks_id ON tasks(id);")

		if !HasTable(conn, "ridx") { // Workaround for non-accepted CREATE VIRTUAL TABLE IF NOT EXISTS
			MustExec(conn, "CREATE VIRTUAL TABLE ridx USING fts3(id TEXT, title_field TEXT, text_field TEXT);")
		}

		MustExec(conn, "CREATE TABLE IF NOT EXISTS columns(id TEXT, name TEXT, value TEXT, FOREIGN KEY (id) REFERENCES tasks (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED)")
		MustExec(conn, "CREATE INDEX IF NOT EXISTS columns_id ON columns(id);")

		MustExec(conn, "CREATE TABLE IF NOT EXISTS saved_searches(name TEXT, value TEXT);")
	}

	MustExec(conn, "CREATE TABLE IF NOT EXISTS settings(name TEXT UNIQUE, value TEXT);")
	MustExec(conn, "INSERT OR IGNORE INTO settings(name, value) VALUES (\"timezone\", \"0\");")
	MustExec(conn, "INSERT OR IGNORE INTO settings(name, value) VALUES (\"theme\", \"tlist.css\");")
	MustExec(conn, "INSERT OR IGNORE INTO settings(name, value) VALUES (\"setup\", \"\");")
	MustExec(conn, "INSERT OR IGNORE INTO settings(name, value) VALUES (\"defaultsorttime\", \"0\");")

	MustExec(conn, "CREATE TABLE IF NOT EXISTS errorlog(timestamp TEXT, message TEXT);")

	MustExec(conn, "CREATE TABLE IF NOT EXISTS private_settings(name TEXT UNIQUE, value TEXT);")
	MustExec(conn, "INSERT OR IGNORE INTO private_settings(name, value) VALUES (\"enable_lua_execution_limit\", \"1\")")

	tasklist := &Tasklist{filename, conn, MakeLuaState(), &LuaFlags{}, &sync.Mutex{}, 1, time.Now().Unix(), true, ""}

	if tasklist.GetPrivateSetting("enable_lua_execution_limit") == "0" {
		Logf(INFO, "Tasklist '%s' runs without lua execution limits", filename)
		tasklist.executionLimitEnabled = false
	}

	tasklist.RunTimedTriggers()
	tasklist.MustExec("PRAGMA foreign_keys = ON;")
	tasklist.MustExec("PRAGMA synchronous = OFF;") // makes inserts many many times faster

	// executing setup code
	setupCode := tasklist.GetSetting("setup")
	if setupCode != "" {
		tasklist.DoString(setupCode, nil) // error is ignored, it will be logged
	}

	return tasklist
}

func (tl *Tasklist) Truncate() {
	tl.MustExec("DELETE FROM columns")
	tl.MustExec("DELETE FROM tasks")
	tl.MustExec("DELETE FROM ridx")
	tl.MustExec("DELETE FROM saved_searches")
	tl.MustExec("DELETE FROM errorlog")
}

func OpenOrCreate(filename string) *Tasklist {
	if !enabledCaching {
		return internalTasklistOpenOrCreate(filename)
	}

	tasklistCacheMutex.Lock()
	defer tasklistCacheMutex.Unlock()

	if r, ok := tasklistCache[filename]; ok && r != nil {
		r.refs++
		r.RunTimedTriggers() // Must run timed triggers anyways
		return r
	}

	Logf(INFO, "Opening new connection to: %s\n", filename)

	r := internalTasklistOpenOrCreate(filename)
	tasklistCache[filename] = r
	return r
}

func (tasklist *Tasklist) Close() {
	if !enabledCaching {
		tasklist.conn.Close()
		tasklist.luaState.Close()
		return
	}

	tasklistCacheMutex.Lock()
	defer tasklistCacheMutex.Unlock()

	tasklist.refs--

	if time.Now().Unix()-tasklist.timestamp < 6*60*60 {
		return
	}
	if tasklist.refs != 0 {
		Logf(ERROR, "Couldn't close stale connection to %s, active connections %d\n", tasklist.filename, tasklist.refs)
		return
	}

	Logf(INFO, "Closing connection to %s\n", tasklist.filename)

	tasklist.conn.Close()
	tasklist.luaState.Close()
	tasklistCache[tasklist.filename] = nil
}

func (tasklist *Tasklist) Exists(id string) bool {
	stmt, err := tasklist.conn.Prepare("SELECT id FROM tasks WHERE id = ?")
	Must(err)
	defer stmt.Finalize()
	Must(stmt.Exec(id))

	hasnext := stmt.Next()
	Log(DEBUG, "Existence of ", id, " ", hasnext)

	return hasnext
}

func MakeRandomString(size int) string {
	var buf []byte = make([]byte, size)

	_, err := io.ReadFull(rand.Reader, buf)
	Must(err)

	var encbuf []byte = make([]byte, base64.StdEncoding.EncodedLen(len(buf)))
	base64.StdEncoding.Encode(encbuf, buf)

	return strings.Replace(strings.Replace(strings.Replace(string(encbuf), "+", "_", -1), "/", "_", -1), "=", "_", -1)
}

func (tasklist *Tasklist) MakeRandomId() string {
	id := MakeRandomString(6)

	exists := tasklist.Exists(id)

	if exists {
		return tasklist.MakeRandomId()
	}

	return id
}

func (tasklist *Tasklist) Remove(id string) {
	tasklist.MustExec("DELETE FROM tasks WHERE id = ?", id)
	tasklist.MustExec("DELETE FROM ridx WHERE id = ?", id)
}

func FormatTriggerAtForAdd(e *Entry) string {
	var triggerAtString string
	if e.TriggerAt() != nil {
		triggerAtString = e.TriggerAt().Format("2006-01-02 15:04:05")
	} else {
		triggerAtString = ""
	}

	return triggerAtString
}

func (tasklist *Tasklist) Quote(in string) string {
	stmt, _ := tasklist.conn.Prepare("SELECT quote(?)")
	defer stmt.Finalize()
	stmt.Exec(in)
	stmt.Next()
	var r string
	stmt.Scan(&r)
	return r
}

func (tl *Tasklist) CloneEntry(entry *Entry) *Entry {
	cols := make(Columns)
	for k, v := range entry.Columns() {
		cols[k] = v
	}
	var triggerAt time.Time = *(entry.TriggerAt())
	return MakeEntry(tl.MakeRandomId(), entry.Title(), entry.Text(), entry.Priority(), &triggerAt, entry.Sort(), cols)
}

func (tasklist *Tasklist) addColumns(e *Entry) {
	for k, v := range e.Columns() {
		Logf(DEBUG, "Adding column %s\n", k)
		tasklist.MustExec("INSERT INTO columns(id, name, value) VALUES (?, ?, ?)", e.Id(), k, v)
	}
}

func (tasklist *Tasklist) Add(e *Entry) {
	triggerAtString := FormatTriggerAtForAdd(e)

	tasklist.WithTransaction(func() {
		priority := e.Priority()
		tasklist.MustExec("INSERT INTO tasks(id, title_field, text_field, priority, trigger_at_field, sort) VALUES (?, ?, ?, ?, ?, ?)", e.Id(), e.Title(), e.Text(), priority.ToInteger(), triggerAtString, e.Sort())
		tasklist.MustExec("INSERT INTO ridx(id, title_field, text_field) VALUES (?, ?, ?)", e.Id(), e.Title(), e.Text())
		tasklist.addColumns(e)
	})

	if CurrentLogLevel <= DEBUG {
		exists := tasklist.Exists(e.Id())
		Log(DEBUG, "Existence check:", exists)
	}

	Log(DEBUG, "Add finished!")
}

func (tasklist *Tasklist) LogError(error string) {
	tasklist.MustExec("INSERT INTO errorlog(timestamp, message) VALUES(?, ?)", time.Now().Unix(), error)
	Logf(INFO, "error while executing lua function: %s\n", error)
}

func (tl *Tasklist) RemoveSaveSearch(name string) {
	tl.MustExec("DELETE FROM saved_searches WHERE name = ?", name)
}

func (tl *Tasklist) SaveSearch(name string, query string) {
	tl.WithTransaction(func() {
		tl.RemoveSaveSearch(name)
		tl.MustExec("INSERT INTO saved_searches(name, value) VALUES(?, ?)", name, query)
	})
}

func (tasklist *Tasklist) Update(e *Entry, simpleUpdate bool) {
	triggerAtString := FormatTriggerAtForAdd(e)
	priority := e.Priority()

	tasklist.WithTransaction(func() {
		tasklist.MustExec("UPDATE tasks SET title_field = ?, text_field = ?, priority = ?, trigger_at_field = ?, sort = ? WHERE id = ?", e.Title(), e.Text(), priority.ToInteger(), triggerAtString, e.Sort(), e.Id())
		if !simpleUpdate {
			tasklist.MustExec("UPDATE ridx SET title_field = ?, text_field = ? WHERE id = ?", e.Title(), e.Text(), e.Id())
			tasklist.MustExec("DELETE FROM columns WHERE id = ?", e.Id())
			tasklist.addColumns(e)
		}
	})

	Log(DEBUG, "Update finished!")
}

func StatementScan(stmt *sqlite.Stmt, hasCols bool) (*Entry, error) {
	var priority_num int
	var trigger_str, id, title, text, sort, columns string
	var scanerr error
	if hasCols {
		scanerr = stmt.Scan(&id, &title, &text, &priority_num, &trigger_str, &sort, &columns)
	} else {
		scanerr = stmt.Scan(&id, &title, &text, &priority_num, &trigger_str, &sort)
	}
	triggerAt, _ := ParseDateTime(trigger_str, 0)
	priority := Priority(priority_num)

	if scanerr != nil {
		return nil, scanerr
	}

	Logf(DEBUG, "Reading columns: %v [%s]\n", hasCols, columns)

	cols := make(Columns)
	if hasCols {
		pieces := strings.Split(columns, "\u001f")
		for i := 0; i+1 < len(pieces); i += 2 {
			Logf(DEBUG, "   col: [%s] [%s]\n", pieces[0], pieces[1])
			cols[pieces[i]] = pieces[i+1]
		}
	}

	Logf(DEBUG, "Columns are: %v\n", cols)

	return MakeEntry(id, title, text, priority, triggerAt, sort, cols), nil
}

func (tl *Tasklist) Get(id string) *Entry {
	stmt, serr := tl.conn.Prepare(SELECT_HEADER + "WHERE tasks.id = ? GROUP BY tasks.id")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec(id))

	if !stmt.Next() {
		panic(fmt.Sprintf("Couldn't find request entry at Tasklist.Get"))
	}

	entry, err := StatementScan(stmt, true)
	Must(err)

	entry.SetId(id)

	return entry
}

func (tl *Tasklist) GetListEx(stmt *sqlite.Stmt, code string, incsub bool) ([]*Entry, error) {
	var err error

	if code != "" {
		tl.luaState.CheckStack(1)
		tl.luaState.PushNil()
		tl.luaState.SetGlobal(SEARCHFUNCTION)
		tl.luaState.DoString(fmt.Sprintf("function %s()\n%s\nend", SEARCHFUNCTION, code))
		tl.luaState.GetGlobal(SEARCHFUNCTION)
		if tl.luaState.IsNil(-1) {
			tl.LogError("Syntax error in search function definition")
			code = ""
			err = &LuaIntError{"Syntax error in search function definition"}
		}
		tl.luaState.Pop(1)
	}

	v := []*Entry{}
	for stmt.Next() {
		entry, scanerr := StatementScan(stmt, true)
		Must(scanerr)

		if code != "" {
			if cerr := tl.CallLuaFunction(SEARCHFUNCTION, entry); cerr != nil {
				err = cerr
			}

			if tl.luaFlags.remove {
				tl.Remove(entry.Id())
			}

			if tl.luaFlags.persist {
				if !tl.luaFlags.remove && tl.luaFlags.cursorEdited {
					tl.Update(entry, false)
				}
				if tl.luaFlags.cursorCloned {
					newentry := GetEntryFromLua(tl.luaState, CURSOR, "%internal%")
					tl.Add(newentry)
				}
			}

			if tl.luaFlags.filterOut {
				continue
			}
		}

		if !incsub {
			skip := false
			for k, _ := range entry.Columns() {
				if strings.HasPrefix(k, "sub/") {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
		}

		v = append(v, entry)
	}
	return v, err
}

func (tl *Tasklist) Retrieve(theselect, code string, incsub bool) ([]*Entry, error) {
	stmt, serr := tl.conn.Prepare(theselect)
	Must(serr)
	defer stmt.Finalize()
	serr = stmt.Exec()
	Must(serr)

	return tl.GetListEx(stmt, code, incsub)
}

func (tl *Tasklist) RetrieveErrors() []*ErrorEntry {
	stmt, serr := tl.conn.Prepare("SELECT timestamp, message FROM errorlog ORDER BY timestamp DESC LIMIT 200")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec())

	r := make([]*ErrorEntry, 0)
	for stmt.Next() {
		var timestamp int64
		var message string
		Must(stmt.Scan(&timestamp, &message))
		r = append(r, &ErrorEntry{time.Unix(timestamp, 0), message})
	}

	return r
}

type ExplainEntry struct {
	Addr               string
	Opcode             string
	P1, P2, P3, P4, P5 string
	Comment            string
}

func (tl *Tasklist) ExplainRetrieve(theselect string) []*ExplainEntry {
	stmt, serr := tl.conn.Prepare(theselect)
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec())

	r := make([]*ExplainEntry, 0)
	for stmt.Next() {
		ee := &ExplainEntry{}
		Must(stmt.Scan(&(ee.Addr), &(ee.Opcode),
			&(ee.P1), &(ee.P2), &(ee.P3), &(ee.P4), &(ee.P5),
			&(ee.Comment)))
		r = append(r, ee)
	}

	return r
}

func (tl *Tasklist) GetSavedSearches() []string {
	v := make([]string, 0)

	stmt, serr := tl.conn.Prepare("SELECT name FROM saved_searches;")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec())

	for stmt.Next() {
		var name string
		Must(stmt.Scan(&name))
		v = append(v, name)
	}

	return v
}

func (tl *Tasklist) GetSavedSearch(name string) string {
	stmt, serr := tl.conn.Prepare("SELECT value FROM saved_searches WHERE name = ?")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec(name))
	if !stmt.Next() {
		return ""
	}
	var value string
	Must(stmt.Scan(&value))
	return value
}

func (tl *Tasklist) GetSetting(name string) string {
	stmt, serr := tl.conn.Prepare("SELECT value FROM settings WHERE name = ?;")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec(name))

	if !stmt.Next() {
		return ""
	}

	var value string
	Must(stmt.Scan(&value))
	return value
}

func (tl *Tasklist) GetPrivateSetting(name string) string {
	stmt, serr := tl.conn.Prepare("SELECT value FROM private_settings WHERE name = ?;")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec(name))

	if !stmt.Next() {
		return ""
	}

	var value string
	Must(stmt.Scan(&value))
	return value
}

func (tl *Tasklist) GetTimezone() int {
	r, _ := strconv.Atoi(tl.GetSetting("timezone"))
	return r
}

func (tl *Tasklist) GetSettings() (r map[string]string) {
	r = make(map[string]string)
	stmt, serr := tl.conn.Prepare("SELECT name, value FROM settings")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec())

	for stmt.Next() {
		var name, value string
		Must(stmt.Scan(&name, &value))
		r[name] = value
	}

	return
}

func (tl *Tasklist) GetTags() []string {
	r := make([]string, 0)

	stmt, serr := tl.conn.Prepare("SELECT DISTINCT name FROM columns WHERE value = ''")
	Must(serr)
	defer stmt.Finalize()
	Must(stmt.Exec())

	for stmt.Next() {
		name := ""
		Must(stmt.Scan(&name))
		if !(strings.HasPrefix(name, "sub/")) {
			r = append(r, name)
		}
	}

	return r
}

func (tl *Tasklist) SetSetting(name, value string) {
	tl.MustExec("INSERT OR REPLACE INTO settings(name, value) VALUES (?, ?);", name, value)
}

func (tl *Tasklist) SetPrivateSetting(name, value string) {
	tl.MustExec("INSERT OR REPLACE INTO private_settings(name, value) VALUES (?, ?);", name, value)
}

func (tl *Tasklist) SetSettings(settings map[string]string) {
	for k, v := range settings {
		Logf(INFO, "Saving %s to %s\n", v, k)
		tl.MustExec("INSERT OR REPLACE INTO settings(name, value) VALUES (?, ?);", k, v)
	}
}

func (tl *Tasklist) RenameTag(src, dst string) {
	if isQuickTagStart(rune(src[0])) {
		src = src[1:len(src)]
	}
	if isQuickTagStart(rune(dst[0])) {
		dst = dst[1:len(dst)]
	}
	tl.MustExec("UPDATE columns SET name = ? WHERE name = ?", dst, src)
}

func (tl *Tasklist) RunTimedTriggers() {
	stmt, serr := tl.conn.Prepare(SELECT_HEADER + "WHERE tasks.trigger_at_field < ? AND tasks.priority = ? GROUP BY id")
	Must(serr)
	defer stmt.Finalize()

	Must(stmt.Exec(time.Now().Format("2006-01-02 15:04:05"), TIMED))

	for stmt.Next() {
		entry, scanerr := StatementScan(stmt, true)
		Must(scanerr)

		if entry.TriggerAt() == nil {
			continue
		} // why was this retrieved?

		update := true
		checkFreq := true

		if triggerCode, ok := entry.ColumnOk("!trigger"); ok {
			Logf(INFO, "Triggering: %s %s with trigger function\n", entry.Id(), entry.TriggerAt())
			tl.DoString(triggerCode, entry)

			if tl.luaFlags.remove {
				tl.Remove(entry.Id())
				update = false
			}

			if tl.luaFlags.persist {
				if tl.luaFlags.cursorCloned {
					newentry := GetEntryFromLua(tl.luaState, CURSOR, "%internal%")
					Logf(INFO, "Cloned, the clone id is: %s\n", newentry.Id())
					tl.Add(newentry)
					checkFreq = false
				}

				if !tl.luaFlags.remove && tl.luaFlags.cursorEdited {
					entry.SetPriority(NOW)
					tl.Update(entry, false)
					update = false
				}
			}

		}

		if checkFreq {
			freq := entry.Freq()

			Logf(INFO, "Triggering: %v %v %v\n", entry.Id(), entry.TriggerAt(), freq)

			if freq > 0 {
				tl.Add(entry.NextEntry(tl.MakeRandomId()))
			}
		}

		if update {
			tl.MustExec("UPDATE tasks SET priority = ? WHERE id = ?", NOW, entry.Id())
		}
	}
}

func (tl *Tasklist) UpgradePriority(id string, special bool) Priority {
	entry := tl.Get(id)
	simpleUpdate := entry.UpgradePriority(special)
	tl.Update(entry, simpleUpdate)
	return entry.Priority()
}

type Statistic struct {
	Name, Link       string
	Total            int
	Now, Later, Done int
	Timed            int
	Notes, Sticky    int
}

func (tl *Tasklist) CountCategoryItems(cat string) (r int) {
	stmt, err := tl.conn.Prepare("SELECT count(distinct id) FROM columns WHERE name = ?")
	Must(err)
	defer stmt.Finalize()
	Must(stmt.Exec("sub/" + cat))

	if stmt.Next() {
		stmt.Scan(&r)
	}

	return
}

func (tl *Tasklist) GetStatistic(tag string) *Statistic {
	var stmt *sqlite.Stmt
	var err error
	if tag == "" {
		stmt, err = tl.conn.Prepare("SELECT priority, count(priority) FROM tasks GROUP BY priority")
	} else {
		stmt, err = tl.conn.Prepare("SELECT priority, count(priority) FROM tasks WHERE id IN (SELECT id FROM columns WHERE name = ?) GROUP BY priority")
	}
	Must(err)
	defer stmt.Finalize()
	if tag == "" {
		Must(stmt.Exec())
	} else {
		Must(stmt.Exec(tag))
	}

	name := "#" + tag
	link := name
	if tag == "" {
		name = "Any"
		link = ""
	}

	r := &Statistic{name, link, 0, 0, 0, 0, 0, 0, 0}

	for stmt.Next() {
		var priority, count int

		stmt.Scan(&priority, &count)

		switch Priority(priority) {
		case STICKY:
			r.Sticky += count
		case NOTES:
			r.Notes += count
		case NOW:
			r.Now += count
		case LATER:
			r.Later += count
		case DONE:
			r.Done += count
		case TIMED:
			r.Timed += count
		default:
			r.Total += count
		}
	}

	r.Total += r.Sticky + r.Notes + r.Now + r.Later + r.Done + r.Timed

	return r
}

func (tl *Tasklist) GetStatistics() []*Statistic {
	r := make([]*Statistic, 0)

	r = append(r, tl.GetStatistic(""))

	for _, tag := range tl.GetTags() {
		r = append(r, tl.GetStatistic(tag))
	}

	return r
}

func (tl *Tasklist) ShowReturnValueRequest() bool {
	return tl.luaFlags.showReturnValue
}

func (tl *Tasklist) GetOntology() []OntologyNodeIn {
	var ontology []OntologyNodeIn
	_ = json.Unmarshal([]byte(tl.GetSetting("ontology")), &ontology)
	return ontology
}

type InvOntologyEntry struct {
	cat     string
	parents []string
}

func (ioe *InvOntologyEntry) String() string {
	return fmt.Sprintf("<%s> [%s]", ioe.cat, strings.Join(ioe.parents, ","))
}

var initialHash = regexp.MustCompile("^#")

func InvertOntology(p []string, ontology []OntologyNodeIn, r []InvOntologyEntry) []InvOntologyEntry {
	for _, on := range ontology {
		n := initialHash.ReplaceAllString(on.Data, "")

		pp := make([]string, len(p))
		copy(pp, p)
		r = append(r, InvOntologyEntry{n, pp})

		ph := make([]string, len(p))
		copy(ph, p)
		ph = append(ph, n)

		if on.Children != nil {
			r = InvertOntology(ph, on.Children, r)
		}
	}
	return r
}

type OntoCheckResult int

const (
	DOES_NOT_APPLY = OntoCheckResult(iota)
	MATCH_OK
	MATCH_FAIL
)

func checkEntryOnOntology(debug bool, e *Entry, ioe InvOntologyEntry) (out OntoCheckResult, mismatchParent string) {
	if _, ok := e.columns[ioe.cat]; !ok {
		return DOES_NOT_APPLY, ""
	}

	for _, p := range ioe.parents {
		if _, ok := e.columns[p]; !ok {
			return MATCH_FAIL, p
		}
	}
	return MATCH_OK, ""
}

func (tl *Tasklist) OntoCheck(debug bool) []OntoCheckError {
	if debug {
		fmt.Printf("Retrieving ontology\n")
	}

	errors := []OntoCheckError{}

	io := []InvOntologyEntry{}
	io = InvertOntology([]string{}, tl.GetOntology(), io)

	if debug {
		fmt.Printf("Retrieving full contents\n")
	}
	theselect, _, _, _, _, _, _, perr := tl.ParseSearch("#:w/done", nil)
	Must(perr)
	v, rerr := tl.Retrieve(theselect, "", false)
	Must(rerr)

	if debug {
		fmt.Printf("%d entries loaded\nChecking\n", len(v))
	}

	unk := 0

	for _, entry := range v {
		result := DOES_NOT_APPLY
		failAt := ""
		failWith := ""
		for _, ioe := range io {
			r, failParent := checkEntryOnOntology(debug, entry, ioe)
			if r == MATCH_OK {
				result = MATCH_OK
				break
			} else if r == MATCH_FAIL {
				result = MATCH_FAIL
				if failAt != "" {
					failAt += ","
					failWith = ""
				} else {
					failWith = failParent
				}
				failAt += ioe.cat
				// continue searching for a successful one
			}
		}

		if result == DOES_NOT_APPLY {
			if debug {
				fmt.Printf("No rule found to apply to entry: %s\n", entry.Id())
			}
			unk++
		}

		if result == MATCH_FAIL {
			errors = append(errors, OntoCheckError{entry, failAt, failWith})
		}
	}

	if debug {
		fmt.Printf("Done: %d errors %d unknown\n", len(errors), unk)
	}

	return errors
}

func (tl *Tasklist) CategoryDepth() map[string]int {
	io := InvertOntology([]string{}, tl.GetOntology(), []InvOntologyEntry{})
	appearsAsParent := map[string]bool{}
	r := map[string]int{}
	for _, ioe := range io {
		if _, ok := r[ioe.cat]; ok {
			r[ioe.cat] = 1000
		} else {
			r[ioe.cat] = len(ioe.parents)
		}

		for _, p := range ioe.parents {
			appearsAsParent[p] = true
		}
	}

	for cat, v := range r {
		if v != 0 {
			continue
		}
		if _, ok := appearsAsParent[cat]; ok {
			continue
		}
		r[cat] = 1000
	}

	return r
}

func (tl *Tasklist) GetChildren(id string) []string {
	stmt, serr := tl.conn.Prepare("select id from columns where name = ? order by value asc")
	Must(serr)
	defer stmt.Finalize()
	serr = stmt.Exec("sub/" + id)
	Must(serr)

	r := []string{}
	for stmt.Next() {
		var x string
		Must(stmt.Scan(&x))
		r = append(r, x)
	}
	return r
}

func (tl *Tasklist) UpdateChildren(pid string, childs []string) {
	for i := range childs {
		tl.MustExec("update columns set value = ? where id = ? and name = ?", fmt.Sprintf("%d", i), childs[i], "sub/"+pid)
	}
}

func ontologyRemove(ontology []OntologyNodeIn, d string) ([]OntologyNodeIn, bool) {
	//TODO
	return nil, false
}

func ontologyAdd(ontology []OntologyNodeIn, p, d string) ([]OntologyNodeIn, bool) {
	//TODO
	return nil, false
}
