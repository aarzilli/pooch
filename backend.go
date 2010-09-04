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

func Create(filename string) {
	conn, err := sqlite.Open(filename)
	if err != nil {
		panic(fmt.Sprintf("Unable to open the database: %s", err))
	}

	defer conn.Close()

	err = conn.Exec("CREATE TABLE tasks(id TEXT, title_field TEXT, text_field TEXT, priority INTEGER, repeat_field INTEGER, trigger_at_field DATE, sort TEXT);")
	if err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE TABLE statement in backend.Create function: %s", err))
	}

	err = conn.Exec("CREATE VIRTUAL TABLE ridx USING fts3(id TEXT, title_field TEXT, text_field TEXT);")
	if err != nil {
		panic(fmt.Sprintf("Unable to execute CREATE VIRTUAL TABLE statement in backend.Create function: %s", err))
	}

	return
}

func Open(name string) (tasklist *Tasklist) {
	conn, sqerr := sqlite.Open(name)
	if sqerr != nil {
		panic(fmt.Sprintf("Cannot open tasklist: %s", sqerr.String()))
	}

	tasklist = &Tasklist{name, conn}
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
	Log(DEBUG, "Existence of", id, hasnext)
	
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
	err := tasklist.conn.Exec("DELETE FROM tasks WHERE id = ?", id)
	if err != nil {
		panic(fmt.Sprintf("Could not remove entry %s: %s", id, err.String()))
	}

	err = tasklist.conn.Exec("DELETE FROM ridx WHERE id = ?", id)
	if err != nil {
		panic(fmt.Sprintf("Could not remove enry %s from reversed index: %s", id, err))
	}

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

func (tasklist *Tasklist) Add(e *Entry) {
	triggerAtString := FormatTriggerAtForAdd(e)

	priority := e.Priority()
	freq := e.Freq()
	err := tasklist.conn.Exec("INSERT INTO tasks(id, title_field, text_field, priority, repeat_field, trigger_at_field, sort) VALUES (?, ?, ?, ?, ?, ?, ?)", e.Id(), e.Title(), e.Text(), priority.ToInteger(), freq.ToInteger(), triggerAtString, e.Sort())
	if err != nil {
		panic(fmt.Sprintf("Error executing INSERT statement for Tasklist.Add: %s", err))
	}

	err = tasklist.conn.Exec("INSERT INTO ridx(id, title_field, text_field) VALUES (?, ?, ?)",
		e.Id(), e.Title(), e.Text())
	if err != nil {
		panic(fmt.Sprintf("Error executing INSERT statement for Tasklist.Add (in ridx): %s", err))
	}

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

	err := tasklist.conn.Exec("UPDATE tasks SET title_field = ?, text_field = ?, priority = ?, repeat_field = ?, trigger_at_field = ?, sort = ? WHERE id = ?", e.Title(), e.Text(), priority.ToInteger(), freq.ToInteger(), triggerAtString, e.Sort(), e.Id())
	if err != nil {
		panic(fmt.Sprintf("Error executing UPDATE statement for Tasklist.Update: %s", err))
	}

	err = tasklist.conn.Exec("UPDATE ridx SET title_field = ?, text_field = ? WHERE id = ?",
		e.Title(), e.Text(), e.Id())
	if err != nil {
		panic(fmt.Sprintf("Error executing UPDATE statement on ridx for Tasklist.Update: %s", err))
	}

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

	return MakeEntry(id, title, text, priority, freq, triggerAt, sort), nil
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

		serr2 := tl.conn.Exec("UPDATE tasks SET priority = ? WHERE id = ?", NOW, entry.Id());

		if serr2 != nil {
			panic(fmt.Sprintf("Error executing UPDATE statement in Tasklist.RunTimedTriggers: %s", serr2))
		}
	}

	return
}

func (tl *Tasklist) UpgradePriority(id string, special bool) Priority {
	entry := tl.Get(id)
	entry.UpgradePriority(special)
	tl.Update(entry)
	return entry.Priority()
}
