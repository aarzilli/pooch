/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"os"
	"fmt"
	"time"
	"io"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"sync"
	"strconv"

	"gosqlite.googlecode.com/hg/sqlite"
)

type Tasklist struct {
	filename string
	conn *sqlite.Conn
	mutex *sync.Mutex
	refs int
	timestamp int64
}

var enabledCaching bool = true
var tasklistCache map[string]*Tasklist = make(map[string]*Tasklist)
var tasklistCacheMutex sync.Mutex

func MustExec(conn *sqlite.Conn, stmt string, v...interface{}) {
	must(conn.Exec(stmt, v...))
}

func HasTable(conn *sqlite.Conn, name string) bool {
	stmt, err := conn.Prepare("SELECT name FROM sqlite_master WHERE name = ?")
	must(err)
	defer stmt.Finalize()
	must(stmt.Exec(name))
	return stmt.Next()
}

func (tasklist *Tasklist) MustExec(stmt string, v...interface{}) {
	MustExec(tasklist.conn, stmt, v...)
}

func (tasklist *Tasklist) WithTransaction(f func()) {
	if enabledCaching {
		tasklist.mutex.Lock()
		defer tasklist.mutex.Unlock()
	} else {
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
	}

	f()
}

func internalTasklistOpenOrCreate(filename string) *Tasklist {
	conn, err := sqlite.Open(filename)
	must(err)

	if !HasTable(conn, "settings") { // optimization, if tasks exists do not try to create anything
		MustExec(conn, "CREATE TABLE IF NOT EXISTS tasks(id TEXT PRIMARY KEY, title_field TEXT, text_field TEXT, priority INTEGER, trigger_at_field DATE, sort TEXT);")
		MustExec(conn, "CREATE INDEX IF NOT EXISTS tasks_id ON tasks(id);")

		if !HasTable(conn, "ridx") { // Workaround for non-accepted CREATE VIRTUAL TABLE IF NOT EXISTS
			MustExec(conn, "CREATE VIRTUAL TABLE ridx USING fts3(id TEXT, title_field TEXT, text_field TEXT);")
		}
		
		MustExec(conn, "CREATE TABLE IF NOT EXISTS columns(id TEXT, name TEXT, value TEXT, FOREIGN KEY (id) REFERENCES tasks (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED)")
		MustExec(conn, "CREATE INDEX IF NOT EXISTS columns_id ON columns(id);")
		
		MustExec(conn, "CREATE TABLE IF NOT EXISTS saved_searches(name TEXT, value TEXT);")

		if !HasTable(conn, "settings") {
			MustExec(conn, "CREATE TABLE IF NOT EXISTS settings(name TEXT UNIQUE, value TEXT);")
			MustExec(conn, "INSERT INTO settings(name, value) VALUES (\"timezone\", \"0\");")
			MustExec(conn, "INSERT INTO settings(name, value) VALUES (\"theme\", \"list.css\");")
		}
	}

	tasklist := &Tasklist{filename, conn, &sync.Mutex{}, 1, time.Seconds()}
	tasklist.RunTimedTriggers()
	tasklist.MustExec("PRAGMA foreign_keys = ON;")
	tasklist.MustExec("PRAGMA synchronous = OFF;") // makes inserts many many times faster

	return tasklist
}

func OpenOrCreate(filename string) *Tasklist {
	if !enabledCaching {
		return internalTasklistOpenOrCreate(filename)
	}

	tasklistCacheMutex.Lock()
	defer tasklistCacheMutex.Unlock()

	if r, ok := tasklistCache[filename]; ok && r != nil {
		r.refs++
		r.RunTimedTriggers() // must run timed triggers anyways
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
		return
	}

	tasklistCacheMutex.Lock()
	defer tasklistCacheMutex.Unlock()
	
	tasklist.refs--
	
	if time.Seconds() - tasklist.timestamp < 6 * 60 * 60 { return }
	if tasklist.refs != 0 {
		Logf(ERROR, "Couldn't close stale connection to %s, active connections %d\n", tasklist.filename, tasklist.refs)
		return
	}

	Logf(INFO, "Closing connection to %s\n", tasklist.filename)
	
	tasklist.conn.Close()
	tasklistCache[tasklist.filename] = nil
}

func (tasklist *Tasklist) Exists(id string) bool {
	stmt, err := tasklist.conn.Prepare("SELECT id FROM tasks WHERE id = ?")
	must(err)
	defer stmt.Finalize()
	must(stmt.Exec(id))

	hasnext := stmt.Next()
	Log(DEBUG, "Existence of ", id, " ", hasnext)
	
	return hasnext
}

func MakeRandomString(size int) string {
	var buf []byte = make([]byte, size)
	
	_, err := io.ReadFull(rand.Reader, buf)
	must(err)

	var encbuf []byte = make([]byte, base64.StdEncoding.EncodedLen(len(buf)))
	base64.StdEncoding.Encode(encbuf, buf)

	return strings.Replace(strings.Replace(string(encbuf), "+", "_", -1), "/", "_", -1)
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
		tasklist.addColumns(e);
	})

	if CurrentLogLevel <= DEBUG {
		exists := tasklist.Exists(e.Id())
		Log(DEBUG, "Existence check:", exists)
	}
		
	Log(DEBUG, "Add finished!")
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
			tasklist.addColumns(e);
		}
	})

	Log(DEBUG, "Update finished!")
}

func StatementScan(stmt *sqlite.Stmt, hasCols bool) (*Entry, os.Error) {
	var priority_num int
	var trigger_str, id, title, text, sort, columns string
	var scanerr os.Error
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
		for _, v := range strings.Split(columns, "\n", -1) {
			col := strings.Split(v, ":", 2)
			Logf(DEBUG, "   col: %s\n", col)
			cols[col[0]] = col[1]
		}
	}

	Logf(DEBUG, "Columns are: %v\n", cols)

	return MakeEntry(id, title, text, priority, triggerAt, sort, cols), nil
}

func (tl *Tasklist) Get(id string) *Entry {
	stmt, serr := tl.conn.Prepare("SELECT tasks.id, tasks.title_field, tasks.text_field, tasks.priority, tasks.trigger_at_field, tasks.sort, group_concat(columns.name||':'||columns.value, '\n') FROM tasks NATURAL JOIN columns WHERE tasks.id = ? GROUP BY tasks.id")
	must(serr)
	defer stmt.Finalize()
	must(stmt.Exec(id))

	if (!stmt.Next()) {
		panic(fmt.Sprintf("Couldn't find request entry at Tasklist.Get"))
	}

	entry, err := StatementScan(stmt, true)
	must(err)

	entry.SetId(id)

	return entry
}

func GetListEx(stmt *sqlite.Stmt) []*Entry{
	v := []*Entry{}
	for (stmt.Next()) {
		entry, scanerr := StatementScan(stmt, true)
		must(scanerr)
		v = append(v, entry)
	}
	return v
}

func (tl *Tasklist) Retrieve(theselect, query string) []*Entry {
	stmt, serr := tl.conn.Prepare(theselect)
	must(serr)
	defer stmt.Finalize()

	if query != "" {
		serr = stmt.Exec(query, query)
	} else {
		serr = stmt.Exec()
	}
	must(serr)

	return GetListEx(stmt)
}

func (tl *Tasklist) GetSavedSearches() []string {
	v := make([]string, 0)

	stmt, serr := tl.conn.Prepare("SELECT name FROM saved_searches;")
	must(serr)
	defer stmt.Finalize()
	must(stmt.Exec())

	for stmt.Next() {
		var name string
		must(stmt.Scan(&name))
		v = append(v, name)
	}

	return v
}

func (tl *Tasklist) GetSavedSearch(name string) string {
	stmt, serr := tl.conn.Prepare("SELECT value FROM saved_searches WHERE name = ?")
	must(serr)
	defer stmt.Finalize()
	must(stmt.Exec(name))
	if !stmt.Next() { return "" }
	var value string
	must(stmt.Scan(&value))
	return value
}

func (tl *Tasklist) GetSetting(name string) string {
	stmt, serr := tl.conn.Prepare("SELECT value FROM settings WHERE name = ?;")
	must(serr)
	defer stmt.Finalize()
	must(stmt.Exec(name))

	if !stmt.Next() { return "" }

	var value string
	must(stmt.Scan(&value))
	return value
}

func (tl *Tasklist) GetTimezone() int {
	r, _ := strconv.Atoi(tl.GetSetting("timezone"))
	return r
}

func (tl *Tasklist) GetSettings() (r map[string]string) {
	r = make(map[string]string)
	stmt, serr := tl.conn.Prepare("SELECT name, value FROM settings")
	must(serr)
	defer stmt.Finalize()
	must(stmt.Exec())

	for stmt.Next() {
		var name, value string
		must(stmt.Scan(&name, &value))
		r[name] = value
	}
	
	return
}

func (tl *Tasklist) SetSetting(name, value string) {
	stmt, serr := tl.conn.Prepare("INSERT OR REPLACE INTO settings(name, value) VALUES (?, ?);")
	must(serr)
	defer stmt.Finalize()
	must(stmt.Exec(name, value))
}

func (tl *Tasklist) SetSettings(settings map[string]string) {
	for k, v := range settings {
		Logf(INFO, "Saving %s to %s\n", v, k);
		tl.MustExec("INSERT OR REPLACE INTO settings(name, value) VALUES (?, ?);", k, v)
	}
}

func (tl *Tasklist) GetSubcols(theselect string) []string {
	v := make([]string, 0)

	stmtStr := "SELECT DISTINCT name FROM columns WHERE value = ''"
	if theselect != "" { stmtStr += " AND id IN (" + theselect + ")"}

	Logf(DEBUG, "Select for Subcols: [%s]\n", stmtStr)

	stmt, serr := tl.conn.Prepare(stmtStr)
	must(serr)
	defer stmt.Finalize()
	must(stmt.Exec())

	for stmt.Next() {
		var name string
		must(stmt.Scan(&name))
		v = append(v, name)
	}

	return v
}

func (tl *Tasklist) RenameTag(src, dst string) {
	if isQuickTagStart(src[0]) { src = src[1:len(src)] }
	if isQuickTagStart(dst[0]) { dst = dst[1:len(dst)] }
	tl.MustExec("UPDATE columns SET name = ? WHERE name = ?", dst, src)
}

func (tl *Tasklist) RunTimedTriggers() {
	stmt, serr := tl.conn.Prepare("SELECT tasks.id, tasks.title_field, tasks.text_field, tasks.priority, tasks.trigger_at_field, tasks.sort, group_concat(columns.name||':'||columns.value, '\n') FROM tasks NATURAL JOIN columns WHERE tasks.trigger_at_field < ? AND tasks.priority = ? GROUP BY id")
	must(serr)
	defer stmt.Finalize()

	must(stmt.Exec(time.UTC().Format("2006-01-02 15:04:05"), TIMED))

	for stmt.Next() {
		entry, scanerr := StatementScan(stmt, true)
		must(scanerr)

		if entry.TriggerAt() == nil {
			continue
		}

		if freq := entry.Freq(); freq > 0{
			Log(INFO, "Triggering:", entry.Id(), entry.TriggerAt(), freq);
			tl.Add(entry.NextEntry(tl.MakeRandomId()));
		} else if _, ok := entry.Column("!trigger"); ok {
			Log(INFO, "Triggering:", entry.Id(), entry.TriggerAt(), "with trigger function")
			//TODO: eseguire funzione
		} else {
			Log(INFO, "Triggering:", entry.Id(), entry.TriggerAt())
		}

		Log(DEBUG, "   updating now")

		tl.MustExec("UPDATE tasks SET priority = ? WHERE id = ?", NOW, entry.Id());
	}
}

func (tl *Tasklist) UpgradePriority(id string, special bool) Priority {
	entry := tl.Get(id)
	entry.UpgradePriority(special)
	tl.Update(entry, true)
	return entry.Priority()
}
