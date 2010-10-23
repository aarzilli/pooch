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
	"container/vector"
	"strings"

	"gosqlite.googlecode.com/hg/sqlite"
)

type Tasklist struct {
	filename string
	conn *sqlite.Conn
}

func MustExec(conn *sqlite.Conn, name string, stmt string, v...interface{}) {
	if err := conn.Exec(stmt, v...); err != nil {
		panic(fmt.Sprintf("Error executing %s: %s", name, err))
	}
}

func HasTable(conn *sqlite.Conn, name string) bool {
	stmt, err := conn.Prepare("SELECT name FROM sqlite_master WHERE name = ?")
	if err != nil { panic(fmt.Sprintf("Error while looking for table named %s: %s", name, err)) }
	defer stmt.Finalize()
	if err := stmt.Exec(name); err != nil { panic(fmt.Sprintf("Error while looking for table named %s: %s", name, err)) }
	return stmt.Next()
}

func (tasklist *Tasklist) MustExec(name string, stmt string, v...interface{}) {
	MustExec(tasklist.conn, name, stmt, v...)
}

func (tasklist *Tasklist) WithTransaction(name string, f func()) {
	tasklist.MustExec(fmt.Sprintf("BEGIN TRANSACTION for %s", name), "BEGIN TRANSACTION")
	defer func() {
		if rerr := recover(); rerr != nil {
			tasklist.conn.Exec("ROLLBACK TRANSACTION")
			panic(rerr)
		} else {
			Logf(DEBUG, "Transaction committed\n")
			tasklist.MustExec(fmt.Sprintf("COMMIT TRANSACTION for %s", name), "COMMIT TRANSACTION")
		}
	}()

	f()
}

func OpenOrCreate(filename string) *Tasklist {
	conn, err := SqliteCachedOpen(filename)
	if err != nil {
		panic(fmt.Sprintf("Unable to open the database: %s", err))
	}

	if err = conn.Exec("CREATE TABLE IF NOT EXISTS tasks(id TEXT PRIMARY KEY, title_field TEXT, text_field TEXT, priority INTEGER, repeat_field INTEGER, trigger_at_field DATE, sort TEXT);"); err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE TABLE statement in backend.Create function: %s", err))
	}

	if !HasTable(conn, "ridx") { // Workaround for non-accepted CREATE VIRTUAL TABLE IF NOT EXISTS
		if err := conn.Exec("CREATE VIRTUAL TABLE ridx USING fts3(id TEXT, title_field TEXT, text_field TEXT);"); err != nil {
			panic(fmt.Sprintf("Unable to execute CREATE VIRTUAL TABLE statement in backend.Create function: %s", err))
		}
	}

	if err := conn.Exec("CREATE TABLE IF NOT EXISTS columns(id TEXT, name TEXT, value TEXT, FOREIGN KEY (id) REFERENCES tasks (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED)"); err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE TABLE (for columns) statement in backend.Create function: %s", err))
	}

	tasklist := &Tasklist{filename, conn}
	tasklist.RunTimedTriggers()
	tasklist.MustExec("PRAGMA on tasklist.Open", "PRAGMA foreign_keys = ON;")

	return tasklist
}

func Port(filename, tag string) {
	conn, err := SqliteCachedOpen(filename)
	if err != nil { panic(fmt.Sprintf("Unable to open the database: %s", err)) }
	defer conn.Close()

	if err := conn.Exec("ALTER TABLE tasks RENAME TO tasks_bku"); err != nil {
		panic(fmt.Sprintf("Unable to execute ALTER TABLE statement in backend.Create function: %s", err))
	}

	if err := conn.Exec("CREATE TABLE tasks(id TEXT PRIMARY KEY, title_field TEXT, text_field TEXT, priority INTEGER, repeat_field INTEGER, trigger_at_field DATE, sort TEXT);"); err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE TABLE statement in backend.Create function: %s", err))
	}

	if err := conn.Exec("INSERT INTO tasks SELECT * FROM tasks_bku"); err != nil {
		panic(fmt.Sprintf("Unable to execute INSERT INTO (tasks copy) statement in backend.Create function: %s", err))
	}

	if err := conn.Exec("DROP TABLE tasks_bku"); err != nil {
		panic(fmt.Sprintf("Unable to execute DROP TABLE (tasks copy) statement in backend.Create function: %s", err))
	}

	if err := conn.Exec("CREATE TABLE columns(id TEXT, name TEXT, value TEXT, FOREIGN KEY (id) REFERENCES tasks (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED);"); err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE TABLE statment in backend.Port: %s", err))
	}

	err = conn.Exec("INSERT INTO columns(id, name, value) SELECT id, ?, '' FROM tasks", tag)
	if err != nil { panic(fmt.Sprintf("Unable to execute INSERT INTO statement in backend.Port: %s", err)) }
}

func Open(name string) (tasklist *Tasklist) {
	conn, sqerr := SqliteCachedOpen(name)
	if sqerr != nil {
		panic(fmt.Sprintf("Cannot open tasklist: %s", sqerr.String()))
	}

	tasklist = &Tasklist{name, conn}
	tasklist.RunTimedTriggers()
	tasklist.MustExec("PRAGMA on tasklist.Open", "PRAGMA foreign_keys = ON;")
	
	return
}

func WithOpen(name string, rest func(tl *Tasklist)) {
	tl := Open(name)
	defer tl.Close()
	rest(tl)
}

func (tasklist *Tasklist) Close() {
	SqliteCachedClose(tasklist.conn)
}

func (tasklist *Tasklist) Exists(id string) bool {
	stmt, err := tasklist.conn.Prepare("SELECT id FROM tasks WHERE id = ?")
	if err != nil {
		panic(fmt.Sprintf("Error preparing statement for Exists: %s", err.String()))
	}
	defer stmt.Finalize()
	err = stmt.Exec(id)
	if err != nil {
		panic(fmt.Sprintf("Error executing statement for Exists: %s", err.String()))
	}

	hasnext := stmt.Next()
	Log(DEBUG, "Existence of ", id, " ", hasnext)
	
	return hasnext
}

func MakeRandomString(size int) string {
	var buf []byte = make([]byte, size)
	
	if 	_, err := io.ReadFull(rand.Reader, buf); err != nil {
		panic(fmt.Sprintf("Error generating random string: %s", err))
	}

	var encbuf []byte = make([]byte, base64.StdEncoding.EncodedLen(len(buf)))
	base64.StdEncoding.Encode(encbuf, buf)

	return strings.Replace(string(encbuf), "+", "_", -1)
	
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
	tasklist.MustExec("DELETE for tasklist.Remove", "DELETE FROM tasks WHERE id = ?", id)
	tasklist.MustExec("DELETE for tasklist.Remove (ridx)", "DELETE FROM ridx WHERE id = ?", id)
	return
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
		tasklist.MustExec("INSERT for extra columns for Tasklist.Add", "INSERT INTO columns(id, name, value) VALUES (?, ?, ?)", e.Id(), k, v)
	}
}

func (tasklist *Tasklist) Add(e *Entry) {
	triggerAtString := FormatTriggerAtForAdd(e)

	tasklist.WithTransaction("backend.Add", func() {
		priority := e.Priority()
		freq := e.Freq()
		tasklist.MustExec("INSERT statement for Tasklist.Add", "INSERT INTO tasks(id, title_field, text_field, priority, repeat_field, trigger_at_field, sort) VALUES (?, ?, ?, ?, ?, ?, ?)", e.Id(), e.Title(), e.Text(), priority.ToInteger(), freq.ToInteger(), triggerAtString, e.Sort())
		tasklist.MustExec("INSERT statement for Tasklist.Add (in ridx)", "INSERT INTO ridx(id, title_field, text_field) VALUES (?, ?, ?)", e.Id(), e.Title(), e.Text())
		tasklist.addColumns(e);
		
	})

	if CurrentLogLevel <= DEBUG {
		exists := tasklist.Exists(e.Id())
		Log(DEBUG, "Existence check:", exists)
	}
		
	Log(DEBUG, "Add finished!")
		
	return
}

func (tasklist *Tasklist) Update(e *Entry) {
	triggerAtString := FormatTriggerAtForAdd(e)
	priority := e.Priority()
	freq := e.Freq()

	tasklist.WithTransaction("backend.Update", func() {
		tasklist.MustExec("UPDATE for tasklist.Update", "UPDATE tasks SET title_field = ?, text_field = ?, priority = ?, repeat_field = ?, trigger_at_field = ?, sort = ? WHERE id = ?", e.Title(), e.Text(), priority.ToInteger(), freq.ToInteger(), triggerAtString, e.Sort(), e.Id())
		tasklist.MustExec("UPDATE for tasklist.Update (in ridx)", "UPDATE ridx SET title_field = ?, text_field = ? WHERE id = ?", e.Title(), e.Text(), e.Id())
		tasklist.MustExec("DELETE of columns for tasklist.Update", "DELETE FROM columns WHERE id = ?", e.Id())
		tasklist.addColumns(e);
	})

	Log(DEBUG, "Update finished!")

	return
}

func StatementScan(stmt *sqlite.Stmt, hasCols bool) (*Entry, os.Error) {
	var priority_num int
	var freq_num int
	var trigger_str, id, title, text, sort, columns string
	var scanerr os.Error
	if hasCols {
		scanerr = stmt.Scan(&id, &title, &text, &priority_num, &freq_num, &trigger_str, &sort, &columns)
	} else {
		scanerr = stmt.Scan(&id, &title, &text, &priority_num, &freq_num, &trigger_str, &sort)
	}
	triggerAt, _ := ParseDateTime(trigger_str)
	freq := Frequency(freq_num)
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

	return MakeEntry(id, title, text, priority, freq, triggerAt, sort, cols), nil
}

func (tl *Tasklist) Get(id string) *Entry {
	stmt, serr := tl.conn.Prepare("SELECT tasks.id, tasks.title_field, tasks.text_field, tasks.priority, tasks.repeat_field, tasks.trigger_at_field, tasks.sort, group_concat(columns.name||':'||columns.value, '\n') FROM tasks NATURAL JOIN columns WHERE tasks.id = ? GROUP BY tasks.id")
	if serr != nil {
		panic(fmt.Sprintf("Error preparing SELECT statement for Tasklist.Get: %s", serr.String()))
	}
	defer stmt.Finalize()
	serr = stmt.Exec(id)
	if serr != nil {
		panic(fmt.Sprintf("Error executing SELECT statement for Tasklist.Get: %s", serr.String()))
	}

	if (!stmt.Next()) {
		panic(fmt.Sprintf("Couldn't find request entry at Tasklist.Get"))
	}

	entry, err := StatementScan(stmt, true)
	if err != nil {
		panic(fmt.Sprintf("Error scanning result of SELECT for Tasklist.Get: %s", err.String()))
	}

	entry.SetId(id)

	return entry
}

func GetListEx(stmt *sqlite.Stmt, v *vector.Vector) {
	for (stmt.Next()) {
		entry, scanerr := StatementScan(stmt, true)

		if scanerr != nil {
			panic(fmt.Sprintf("Error scanning results for Tasklist.GetList: %s", scanerr.String()))
		}

		v.Push(entry);
	}
}

func (tl *Tasklist) Retrieve(theselect, query string) (v *vector.Vector) {
	v = new(vector.Vector);

	stmt, serr := tl.conn.Prepare(theselect)
	if serr != nil { panic(fmt.Sprintf("Error preparing SELECT statement [%s] for tasklist.Retrieve: %s", theselect, serr)) }
	defer stmt.Finalize()

	if query != "" {
		serr = stmt.Exec(query, query)
	} else {
		serr = stmt.Exec()
	}
	if serr != nil { panic(fmt.Sprintf("Error executing SELECT statement [%s] for tasklist.Retrieve: %s", theselect, serr)) }

	GetListEx(stmt, v)
	return
}

func (tl *Tasklist) GetSubcols(theselect string) []string {
	var v vector.StringVector

	stmtStr := "SELECT DISTINCT name FROM columns WHERE value = ''"
	if theselect != "" { stmtStr += " AND id IN (" + theselect + ")"}

	Logf(DEBUG, "Select for Subcols: [%s]\n", stmtStr)

	stmt, serr := tl.conn.Prepare(stmtStr)
	if serr != nil { panic(fmt.Sprintf("Error preparing SELECT statement for Tasklist.GetSubcols: %s", serr)) }
	defer stmt.Finalize()
	serr = stmt.Exec()
	if serr != nil { panic(fmt.Sprintf("Error executing SELECT statement for Tasklist.GetSubcols: %s", serr)) }

	for stmt.Next() {
		var name string
		serr = stmt.Scan(&name)
		if serr != nil { panic(fmt.Sprintf("Error scanning for Tsklist.GetSubcols: %s", serr)) }
		v.Push(name)
	}

	return ([]string)(v)
}

func (tl *Tasklist) RunTimedTriggers() {
	//tl.MustExec("PRAGMA on tasklist.Open", "PRAGMA foreign_keys = OFF;")
	stmt, serr := tl.conn.Prepare("SELECT tasks.id, tasks.title_field, tasks.text_field, tasks.priority, tasks.repeat_field, tasks.trigger_at_field, tasks.sort, group_concat(columns.name||':'||columns.value, '\n') FROM tasks NATURAL JOIN columns WHERE tasks.trigger_at_field < ? AND tasks.priority = ? GROUP BY id");
	if serr != nil {
		panic(fmt.Sprintf("Error preparing SELECT statement for Tasklist.RunTimedTriggers: %s", serr))
	}
	defer stmt.Finalize()

	serr = stmt.Exec(time.LocalTime().Format("2006-01-02 15:04:05"), TIMED)
	if serr != nil {
		panic(fmt.Sprintf("Error executing SELECT statement for Tasklist.RunTimedTriggers: %s", serr.String()))
	}

	for stmt.Next() {
		entry, scanerr := StatementScan(stmt, true)
		
		if scanerr != nil {
			panic(fmt.Sprintf("Error scanning results of SELECT statement for Tasklist.RunTimedTriggers: %s", scanerr.String()))
		}

		if entry.TriggerAt() == nil {
			continue
		}

		Log(DEBUG, "Triggering:", entry.Id(), entry.TriggerAt(), entry.Freq());

		if entry.Freq() > 0 {
			tl.Add(entry.NextEntry(tl.MakeRandomId()));
		}

		Log(DEBUG, "   updating now")

		tl.MustExec("UPDATE for Tasklist.RunTimedTriggers", "UPDATE tasks SET priority = ? WHERE id = ?", NOW, entry.Id());
	}
	
	tl.MustExec("PRAGMA on tasklist.Open", "PRAGMA foreign_keys = ON;")
}

func (tl *Tasklist) UpgradePriority(id string, special bool) Priority {
	entry := tl.Get(id)
	entry.UpgradePriority(special)
	tl.Update(entry)
	return entry.Priority()
}
