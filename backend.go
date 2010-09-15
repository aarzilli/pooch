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

func (tasklist *Tasklist) MustExec(name string, stmt string, v...interface{}) {
	if err := tasklist.conn.Exec(stmt, v); err != nil {
		panic(fmt.Sprintf("Error executing %s: %s", name, err))
	}
}

func (tasklist *Tasklist) WithTransaction(name string, f func()) {
	tasklist.MustExec(fmt.Sprintf("BEGIN TRANSACTION for %s", name), "BEGIN TRANSACTION")
	defer func() {
		if rerr := recover(); rerr != nil {
			tasklist.conn.Exec("ROLLBACK TRANSACTION")
			panic(rerr)
		} else {
			tasklist.MustExec(fmt.Sprintf("COMMIT TRANSACTION for %s", name), "COMMIT TRANSACTION")
		}
	}()

	f()
}

func Create(filename string) {
	conn, err := sqlite.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("Unable to open the database: %s", err))
	}

	defer conn.Close()

	if err = conn.Exec("CREATE TABLE tasks(id TEXT PRIMARY KEY, title_field TEXT, text_field TEXT, priority INTEGER, repeat_field INTEGER, trigger_at_field DATE, sort TEXT);"); err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE TABLE statement in backend.Create function: %s", err))
	}

	if err := conn.Exec("CREATE VIRTUAL TABLE ridx USING fts3(id TEXT, title_field TEXT, text_field TEXT);"); err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE VIRTUAL TABLE statement in backend.Create function: %s", err))
	}

	if err := conn.Exec("CREATE TABLE columns(id TEXT, name TEXT, value TEXT, FOREIGN KEY (id) REFERENCES tasks (id) ON DELETE CASCADE DEFERRABLE INITIALLY DEFERRED)"); err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE TABLE (for columns) statement in backend.Create function: %s", err))
	}

	return
}

func Open(name string) (tasklist *Tasklist) {
	conn, sqerr := sqlite.Open(name)
	if sqerr != nil {
		panic(fmt.Sprintf("Cannot open tasklist: %s", sqerr.String()))
	}

	tasklist = &Tasklist{name, conn}
	tasklist.MustExec("PRAGMA on tasklist.Open", "PRAGMA foreign_keys = ON;")
	tasklist.RunTimedTriggers()
	
	return
}

func (tasklist *Tasklist) Close() {
	Log(DEBUG, "Closing connection")
	err := tasklist.conn.Close()
	if err != nil {
		panic(fmt.Sprintf("Couldn't close connection: %s", err.String()))
	}
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

func (tasklist *Tasklist) MakeRandomId() string {
	var buf []byte = make([]byte, 6)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		panic(fmt.Sprintf("Error generating random id: %s", err))
	}

	var encbuf []byte = make([]byte, base64.StdEncoding.EncodedLen(len(buf)))
	base64.StdEncoding.Encode(encbuf, buf)

	id := strings.Replace(string(encbuf), "+", "_", -1)

	exists := tasklist.Exists(id)
	
	if exists {
		return tasklist.MakeRandomId()
	}
	
	return id
}

func (tasklist *Tasklist) Remove(id string) {
	tasklist.MustExec("DELETE for tasklist.Remove", "DELETE FROM tasks WHERE id = ?", id)
	tasklist.MustExec("DELETE for tasklist.Remove (ridx)", "DELETE FROM ridx WHERE id = ?", id)
	//TODO: remove associated columns
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
		
		if CurrentLogLevel <= DEBUG {
			exists := tasklist.Exists(e.Id())
			Log(DEBUG, "Existence check:", exists)
		}
	})
		
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


func StatementScan(stmt *sqlite.Stmt) (*Entry, os.Error) {
	var priority_num int
	var freq_num int
	var trigger_str, id, title, text, sort string
	scanerr := stmt.Scan(&id, &title, &text, &priority_num, &freq_num, &trigger_str, &sort)
	triggerAt, _ := ParseDateTime(trigger_str)
	freq := Frequency(freq_num)
	priority := Priority(priority_num)

	if scanerr != nil {
		return nil, scanerr
	}

	//TODO: read columns here
	cols := make(Columns)

	return MakeEntry(id, title, text, priority, freq, triggerAt, sort, cols), nil
}

func (tl *Tasklist) Get(id string) *Entry {
	stmt, serr := tl.conn.Prepare("SELECT id, title_field, text_field, priority, repeat_field, trigger_at_field, sort FROM tasks WHERE id = ?")
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

	entry, err := StatementScan(stmt)
	if err != nil {
		panic(fmt.Sprintf("Error scanning result of SELECT for Tasklist.Get: %s", err.String()))
	}

	entry.SetId(id)

	return entry
}

func GetListEx(stmt *sqlite.Stmt, v *vector.Vector) {

	for (stmt.Next()) {
		entry, scanerr := StatementScan(stmt)

		if scanerr != nil {
			panic(fmt.Sprintf("Error scanning results for Tasklist.GetList: %s", scanerr.String()))
		}

		v.Push(entry);
	}
}

func (tl *Tasklist) GetList(includeDone bool) (v *vector.Vector) {
	v = new(vector.Vector);

	stmtStr := "SELECT id, title_field, text_field, priority, repeat_field, trigger_at_field, sort FROM tasks";

	if !includeDone {
		stmtStr += " WHERE priority <> 5"
	}

	stmtStr += " ORDER BY priority, trigger_at_field ASC, sort DESC";
	
	stmt, serr := tl.conn.Prepare(stmtStr);
	if serr != nil {
		panic(fmt.Sprintf("Error preparing SELECT statement for Tasklist.GetList: %s", serr))
	}
	defer stmt.Finalize()
	
	serr = stmt.Exec()
	if serr != nil {
		panic(fmt.Sprintf("Error executing SELECT statement for Tasklist.GetList: %s", serr))
	}

	GetListEx(stmt, v)

	return
}

func (tl *Tasklist) GetEventList(start, end string) (v *vector.Vector) {
	v = new(vector.Vector)

	stmtStr := "SELECT id, title_field, text_field, priority, repeat_field, trigger_at_field, sort FROM tasks WHERE trigger_at_field IS NOT NULL AND trigger_at_field > ? AND trigger_at_field < ?"

	stmt, serr := tl.conn.Prepare(stmtStr)
	if serr != nil {
		panic(fmt.Sprintf("Error preparing SELECT statement for Tasklist.GetEventList: %s", serr))
	}
	defer stmt.Finalize()

	serr = stmt.Exec(start, end)
	if serr != nil {
		panic(fmt.Sprintf("Error executing SELECT statement for Tasklist.GetEventList: %s", serr))
	}

	GetListEx(stmt, v)

	return
}

func (tl *Tasklist) Search(query string) (v *vector.Vector) {
	v = new(vector.Vector);

	stmtStr := "SELECT id, title_field, text_field, priority, repeat_field, trigger_at_field, sort FROM tasks WHERE id IN (SELECT id FROM ridx WHERE title_field MATCH ? UNION SELECT id FROM ridx WHERE text_field MATCH ?) ORDER BY priority, sort ASC";

	stmt, serr := tl.conn.Prepare(stmtStr);
	if serr != nil {
		panic(fmt.Sprintf("Error preparing SELECT statement for Tasklist.Search: %s", serr))
	}
	defer stmt.Finalize()
	
	serr = stmt.Exec(query, query)
	if serr != nil {
		panic(fmt.Sprintf("Error executing SELECT statement for Tasklist.GetList: %s", serr.String()))
	}	

	GetListEx(stmt, v)

	return
}

func (tl *Tasklist) RunTimedTriggers() {
	stmt, serr := tl.conn.Prepare("SELECT id, title_field, text_field, priority, repeat_field, trigger_at_field, sort FROM tasks WHERE trigger_at_field < ? AND priority = ?");
	defer stmt.Finalize()
	if serr != nil {
		panic(fmt.Sprintf("Error preparing SELECT statement for Tasklist.RunTimedTriggers: %s", serr.String()))
	}

	serr = stmt.Exec(time.LocalTime().Format("2006-01-02 15:04:05"), TIMED)
	if serr != nil {
		panic(fmt.Sprintf("Error executing SELECT statement for Tasklist.RunTimedTriggers: %s", serr.String()))
	}

	for (stmt.Next()) {
		entry, scanerr := StatementScan(stmt)
		
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

	return
}

func (tl *Tasklist) UpgradePriority(id string, special bool) Priority {
	entry := tl.Get(id)
	entry.UpgradePriority(special)
	tl.Update(entry)
	return entry.Priority()
}
