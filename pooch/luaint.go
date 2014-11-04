/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
*/

package pooch

import (
	"errors"
	"fmt"
	"github.com/aarzilli/golua/lua"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var CURSOR string = "cursor"
var TASKLIST string = "tasklist"
var SEARCHFUNCTION string = "searchfn"
var LUA_EXECUTION_LIMIT = 1000
var RUN_ARGUMENTS_VAR = "args"

type LuaIntError struct {
	message string
}

func (le *LuaIntError) Error() string {
	return le.message
}

type LuaFlags struct {
	cursorEdited bool // the original, introduced cursor, was modified
	cursorCloned bool // the cursor was cloned, creating a new entry
	persist      bool // changes are persisted
	filterOut    bool // during search filters out the current result
	remove       bool // removes the current entry

	freeCursor      bool // function is free of moving the cursor around
	showReturnValue bool // show return value of this function

	objects []interface{}
}

func (tl *Tasklist) ResetLuaFlags() {
	tl.luaFlags.cursorEdited = false
	tl.luaFlags.cursorCloned = false
	tl.luaFlags.persist = false
	tl.luaFlags.filterOut = false
	tl.luaFlags.remove = false
	tl.luaFlags.freeCursor = false
	tl.luaFlags.showReturnValue = false
}

func (tl *Tasklist) SetEntryInLua(name string, entry *Entry) {
	rawptr := tl.luaState.NewUserdata(uintptr(unsafe.Sizeof(entry)))
	var ptr **Entry = (**Entry)(rawptr)
	*ptr = entry
	tl.luaState.SetGlobal(name)
}

func (tl *Tasklist) SetTasklistInLua() {
	rawptr := tl.luaState.NewUserdata(uintptr(unsafe.Sizeof(tl)))
	var ptr **Tasklist = (**Tasklist)(rawptr)
	*ptr = tl
	tl.luaState.SetGlobal(TASKLIST)
}

func GetTasklistFromLua(L *lua.State) *Tasklist {
	L.CheckStack(1)
	L.GetGlobal(TASKLIST)
	rawptr := L.ToUserdata(-1)
	var ptr **Tasklist = (**Tasklist)(rawptr)
	L.Pop(1)
	return *ptr
}

func GetEntryFromLua(L *lua.State, name string, fname string) *Entry {
	L.CheckStack(1)
	L.GetGlobal(name)
	rawptr := L.ToUserdata(-1)
	var ptr **Entry = (**Entry)(rawptr)
	L.Pop(1)
	if ptr == nil {
		panic(errors.New("No cursor set, can not use " + fname))
	}
	return *ptr

}

func LuaIntGetterSetterFunction(fname string, L *lua.State, getter func(tl *Tasklist, entry *Entry) string, setter func(tl *Tasklist, entry *Entry, value string)) int {
	argNum := L.GetTop()

	if argNum == 0 {
		entry := GetEntryFromLua(L, CURSOR, fname)
		tl := GetTasklistFromLua(L)
		L.PushString(getter(tl, entry))
		return 1
	} else if argNum == 1 {
		value := L.ToString(1)
		entry := GetEntryFromLua(L, CURSOR, fname)
		tl := GetTasklistFromLua(L)
		setter(tl, entry, value)
		if !tl.luaFlags.cursorCloned {
			tl.luaFlags.cursorEdited = true
		}
		return 0
	}

	panic(errors.New(fmt.Sprintf("Incorrect number of argoments to %s (only 0 or 1 accepted)", fname)))
	return 0
}

func LuaIntGetterSetterFunctionInt(fname string, L *lua.State, getter func(tl *Tasklist, entry *Entry) int64, setter func(tl *Tasklist, entry *Entry, value int)) int {
	argNum := L.GetTop()

	if argNum == 0 {
		entry := GetEntryFromLua(L, CURSOR, fname)
		tl := GetTasklistFromLua(L)
		L.PushInteger(getter(tl, entry))
		return 1
	} else if argNum == 1 {
		value := L.ToInteger(1)
		entry := GetEntryFromLua(L, CURSOR, fname)
		tl := GetTasklistFromLua(L)
		setter(tl, entry, value)
		if !tl.luaFlags.cursorCloned {
			tl.luaFlags.cursorEdited = true
		}
		return 0
	}

	panic(errors.New(fmt.Sprintf("Incorrect number of argoments to %s (only 0 or 1 accepted)", fname)))
	return 0
}

func LuaIntId(L *lua.State) int {
	return LuaIntGetterSetterFunction("id", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Id() },
		func(tl *Tasklist, entry *Entry, value string) {
			if tl.luaFlags.cursorCloned {
				entry.SetId(value)
			}
		})
}

func LuaIntTitle(L *lua.State) int {
	return LuaIntGetterSetterFunction("title", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Title() },
		func(tl *Tasklist, entry *Entry, value string) { entry.SetTitle(value) })
}

func LuaIntText(L *lua.State) int {
	return LuaIntGetterSetterFunction("text", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Text() },
		func(tl *Tasklist, entry *Entry, value string) { entry.SetText(value) })
}

func LuaIntSortField(L *lua.State) int {
	return LuaIntGetterSetterFunction("sortfield", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Sort() },
		func(tl *Tasklist, entry *Entry, value string) { entry.SetSort(value) })
}

func LuaIntPriority(L *lua.State) int {
	return LuaIntGetterSetterFunction("priority", L,
		func(tl *Tasklist, entry *Entry) string { pr := entry.Priority(); return pr.String() },
		func(tl *Tasklist, entry *Entry, value string) { pr := ParsePriority(value); entry.SetPriority(pr) })
}

func LuaIntWhen(L *lua.State) int {
	return LuaIntGetterSetterFunctionInt("triggerat", L,
		func(tl *Tasklist, entry *Entry) int64 {
			t := entry.TriggerAt()
			if t != nil {
				return int64(t.Unix())
			}
			return 0
		},
		func(tl *Tasklist, entry *Entry, value int) {
			r := time.Unix(int64(value), 0)
			entry.SetTriggerAt(&r)
		})
}

func LuaIntColumn(L *lua.State) int {
	argNum := L.GetTop()

	if argNum == 1 {
		name := L.ToString(1)
		entry := GetEntryFromLua(L, CURSOR, "column()")
		L.PushString(entry.Column(name))
		return 1
	} else if argNum == 2 {
		name := L.ToString(1)
		value := L.ToString(2)
		entry := GetEntryFromLua(L, CURSOR, "column()")
		entry.SetColumn(name, value)
		tl := GetTasklistFromLua(L)
		if !tl.luaFlags.cursorCloned {
			tl.luaFlags.cursorEdited = true
		}
		return 0
	}

	panic(errors.New("Incorrect number of arguments to column (only 1 or 2 accepted)"))
	return 0
}

func luaAssertArgnum(L *lua.State, n int, fname string) {
	if L.GetTop() != n {
		panic(errors.New("Incorrect number of arguments to " + fname))
	}
}

func luaAssertNotFreeCursor(tl *Tasklist, fname string) {
	if tl.luaFlags.freeCursor {
		panic(errors.New("Can not use " + fname + " on free cursor"))
	}
}

func LuaIntRmColumn(L *lua.State) int {
	luaAssertArgnum(L, 1, "rmcolumn()")
	name := L.ToString(1)
	entry := GetEntryFromLua(L, CURSOR, "rmcolumn()")
	entry.RemoveColumn(name)
	tl := GetTasklistFromLua(L)
	if !tl.luaFlags.cursorCloned {
		tl.luaFlags.cursorEdited = true
	}
	return 0
}

func LuaIntFilterOut(L *lua.State) int {
	tl := GetTasklistFromLua(L)
	tl.luaFlags.filterOut = true
	return 0
}

func LuaIntFilterIn(L *lua.State) int {
	tl := GetTasklistFromLua(L)
	tl.luaFlags.filterOut = false
	return 0
}

func LuaIntPersist(L *lua.State) int {
	tl := GetTasklistFromLua(L)
	luaAssertNotFreeCursor(tl, "persist()")
	tl.luaFlags.persist = true
	return 0
}

func LuaIntRemove(L *lua.State) int {
	tl := GetTasklistFromLua(L)
	luaAssertNotFreeCursor(tl, "persist()")
	tl.luaFlags.remove = true
	return 0
}

func LuaIntCloneCursor(L *lua.State) int {
	tl := GetTasklistFromLua(L)
	luaAssertNotFreeCursor(tl, "clone()")
	cursor := GetEntryFromLua(L, CURSOR, "clone()")
	newcursor := tl.CloneEntry(cursor)
	tl.SetEntryInLua(CURSOR, newcursor)
	tl.luaFlags.cursorCloned = true
	return 0
}

func LuaIntWriteCursor(L *lua.State) int {
	Logf(INFO, "Writing cursor")
	L.CheckStack(1)
	tl := GetTasklistFromLua(L)
	luaAssertNotFreeCursor(tl, "writecursor()")
	cursor := GetEntryFromLua(L, CURSOR, "writecursor()")
	tl.Update(cursor, false)
	return 0
}

func LuaIntVisit(L *lua.State) int {
	L.CheckStack(1)
	tl := GetTasklistFromLua(L)
	luaAssertNotFreeCursor(tl, "visit()")
	id := L.ToString(1)
	Logf(DEBUG, "Lua visiting: <%s>\n", id)
	var error interface{} = nil

	{
		defer func() {
			if rerr := recover(); rerr != nil {
				error = rerr
			}
		}()
		cursor := tl.Get(id)
		tl.SetEntryInLua(CURSOR, cursor)
	}

	if error != nil {
		tl.SetEntryInLua(CURSOR, nil)
	}

	return 0
}

func SetTableInt(L *lua.State, name string, value int64) {
	// Remember to check stack for 2 extra locations
	L.PushString(name)
	L.PushInteger(value)
	L.SetTable(-3)
}

func SetTableIntString(L *lua.State, idx int64, value string) {
	L.PushInteger(idx)
	L.PushString(value)
	L.SetTable(-3)
}

func GetTableInt(L *lua.State, name string) int {
	// Remember to check stack for 1 extra location
	L.PushString(name)
	L.GetTable(-2)
	r := L.ToInteger(-1)
	L.Pop(1)
	return r
}

func PushTime(L *lua.State, t time.Time) {
	L.CheckStack(3)
	L.CreateTable(0, 7)

	SetTableInt(L, "year", int64(t.Year()))
	SetTableInt(L, "month", int64(t.Month()))
	SetTableInt(L, "day", int64(t.Day()))
	SetTableInt(L, "weekday", int64(t.Weekday()))
	SetTableInt(L, "hour", int64(t.Hour()))
	SetTableInt(L, "minute", int64(t.Minute()))
	SetTableInt(L, "second", int64(t.Second()))
}

func PushStringVec(L *lua.State, v []string) {
	L.CheckStack(3)
	L.CreateTable(len(v), 0)

	for idx, val := range v {
		SetTableIntString(L, int64(idx+1), val)
	}
}

func LuaIntUTCTime(L *lua.State) int {
	luaAssertArgnum(L, 1, "utctime()")
	timestamp := L.ToInteger(1)
	PushTime(L, time.Unix(int64(timestamp), 0))

	return 1
}

func LuaIntLocalTime(L *lua.State) int {
	luaAssertArgnum(L, 1, "localtime()")
	tl := GetTasklistFromLua(L)
	timezone := tl.GetTimezone()
	timestamp := L.ToInteger(1)

	t := time.Unix(int64(timestamp)+(int64(timezone)*60*60), 0)

	PushTime(L, t)

	return 1
}

func LuaIntTimestamp(L *lua.State) int {
	luaAssertArgnum(L, 1, "timestamp()")

	if !L.IsTable(-1) {
		panic(errors.New("Argoment of timestamp is not a table"))
		return 0
	}

	L.CheckStack(1)

	t := time.Date(GetTableInt(L, "year"), time.Month(GetTableInt(L, "month")), GetTableInt(L, "day"), GetTableInt(L, "hour"), GetTableInt(L, "minute"), GetTableInt(L, "second"), 0, time.Local)

	L.PushInteger(int64(t.Unix()))
	return 1
}

func LuaIntParseDateTime(L *lua.State) int {
	luaAssertArgnum(L, 1, "parsedatetime()")

	L.CheckStack(1)

	input := L.ToString(-1)
	tl := GetTasklistFromLua(L)

	out, _ := ParseDateTime(input, tl.GetTimezone())

	if out != nil {
		L.PushInteger(int64(out.Unix()))
	} else {
		L.PushInteger(0)
	}

	return 1
}

func LuaIntSplit(L *lua.State) int {
	if L.GetTop() < 2 {
		panic(errors.New("Wrong number of arguments to split()"))
		return 0
	}

	instr := L.ToString(1)
	sepstr := L.ToString(2)

	n := -1

	if L.GetTop() == 3 {
		n = L.ToInteger(3)
	}

	if L.GetTop() > 3 {
		panic(errors.New("Wrong number of arguments to split()"))
		return 0
	}

	PushStringVec(L, strings.SplitN(instr, sepstr, n))

	return 1
}

func LuaIntStringFunction(L *lua.State, name string, n int, fn func(tl *Tasklist, argv []string) int) int {
	luaAssertArgnum(L, n, name)

	argv := make([]string, 0)

	for i := 1; i <= n; i++ {
		argv = append(argv, L.ToString(i))
	}
	L.Pop(n)
	L.CheckStack(1)
	tl := GetTasklistFromLua(L)
	return fn(tl, argv)
}

func LuaIntIdQuery(L *lua.State) int {
	return LuaIntStringFunction(L, "idq", 1, func(tl *Tasklist, argv []string) int {
		tl.luaState.PushGoStruct(&SimpleExpr{":id", "=", argv[0], nil, 0, ""})
		return 1
	})
}

func LuaIntTitleQuery(L *lua.State) int {
	return LuaIntStringFunction(L, "titleq", 2, func(tl *Tasklist, argv []string) int {
		tl.luaState.PushGoStruct(&SimpleExpr{":title_field", argv[0], argv[1], nil, 0, ""})
		return 1
	})
}

func LuaIntTextQuery(L *lua.State) int {
	return LuaIntStringFunction(L, "textq", 2, func(tl *Tasklist, argv []string) int {
		tl.luaState.PushGoStruct(&SimpleExpr{":text_field", argv[0], argv[1], nil, 0, ""})
		return 1
	})
}

func LuaIntWhenQuery(L *lua.State) int {
	return LuaIntStringFunction(L, "whenq", 2, func(tl *Tasklist, argv []string) int {
		n, _ := strconv.ParseInt(argv[1], 10, 64)
		t := time.Unix(n, 0)
		tl.luaState.PushGoStruct(&SimpleExpr{":when", argv[0], "", &t, 0, ""})
		return 1
	})
}

func LuaIntSearchQuery(L *lua.State) int {
	return LuaIntStringFunction(L, "searchq", 1, func(tl *Tasklist, argv []string) int {
		tl.luaState.PushGoStruct(&SimpleExpr{":search", "match", argv[0], nil, 0, ""})
		return 1
	})
}

func LuaIntColumnQuery(L *lua.State) int {
	if (L.GetTop() != 1) && (L.GetTop() != 3) {
		panic(errors.New("Wrong number of arguments to columnq"))
		return 0
	}

	name := L.ToString(1)
	op := ""
	value := ""
	if L.GetTop() == 3 {
		op = L.ToString(2)
		value = L.ToString(3)
		L.Pop(3)
	} else {
		L.Pop(1)
	}

	if name[0] == ':' {
		panic(errors.New("Column name can not start with ':'"))
		return 0
	}

	L.CheckStack(1)
	tl := GetTasklistFromLua(L)

	tl.luaState.PushGoStruct(&SimpleExpr{name, op, value, nil, 0, ""})
	return 1
}

func LuaIntPriorityQuery(L *lua.State) int {
	luaAssertArgnum(L, 1, "priorityq()")

	priority := L.ToString(1)

	L.CheckStack(1)
	tl := GetTasklistFromLua(L)

	tl.luaState.PushGoStruct(&SimpleExpr{":priority", "=", priority, nil, ParsePriority(priority), ""})

	return 1
}

func GetQueryObject(tl *Tasklist, i int) Clausable {
	if tl.luaState.IsString(i) {
		parser := NewParser(NewTokenizer(tl.luaState.ToString(i)), tl.GetTimezone())
		se := &SimpleExpr{}
		if parser.ParseSimpleExpression(se) {
			return se
		} else {
			panic(errors.New("Unparsable string in expression"))
			return nil
		}
	}

	ud := tl.luaState.ToGoStruct(i)
	if ud == nil {
		return nil
	}

	if clausable, ok := ud.(Clausable); ok {
		return clausable
	}

	return nil
}

func LuaIntNotQuery(L *lua.State) int {
	luaAssertArgnum(L, 1, "Wrong number of arguments to notq")

	L.CheckStack(1)
	tl := GetTasklistFromLua(L)

	clausable := GetQueryObject(tl, -1)
	if clausable == nil {
		panic(errors.New("Wrong argument type to notq only query objects accepted as arguments"))
		return 0
	}

	tl.luaState.PushGoStruct(&NotExpr{clausable})
	return 1

}

func LuaIntBoolQuery(L *lua.State, operator, name string) int {
	if L.GetTop() < 2 {
		panic(errors.New("Wrong number of arguments to " + name))
		return 0
	}

	L.CheckStack(1)
	tl := GetTasklistFromLua(L)

	r := &BoolExpr{operator, make([]Clausable, 0)}

	for i := 1; i <= L.GetTop(); i++ {
		clausable := GetQueryObject(tl, i)
		if clausable == nil {
			panic(errors.New("Wrong argument type to " + name + " only query objects accepted as arguments"))
			return 0
		}
		r.subExpr = append(r.subExpr, clausable)
	}

	tl.luaState.PushGoStruct(r)
	return 1
}

func LuaIntAndQuery(L *lua.State) int {
	return LuaIntBoolQuery(L, "AND", "andq")
}

func LuaIntOrQuery(L *lua.State) int {
	return LuaIntBoolQuery(L, "OR", "orq")
}

func LuaIntSearch(L *lua.State) int {
	if (L.GetTop() < 1) || (L.GetTop() > 2) {
		panic(errors.New("Wrong number of arguments to search()"))
		return 0
	}

	L.CheckStack(2)
	tl := GetTasklistFromLua(L)

	if !tl.luaFlags.freeCursor {
		panic(errors.New("search() function only available on a free cursor"))
		return 0
	}

	query := L.ToString(1)
	var luaClausable Clausable = nil
	if L.GetTop() == 2 {
		luaClausable = GetQueryObject(tl, 2)
	}

	theselect, _, _, _, _, _, _, perr := tl.ParseSearch(query, luaClausable)
	Must(perr)

	entries, serr := tl.Retrieve(theselect, "", false)
	Must(serr)

	Logf(INFO, "Searching from lua interface <%s> clausable: <%v> yields %d results\n", query, luaClausable, len(entries))

	r := []string{}
	for _, entry := range entries {
		r = append(r, entry.Id())
	}

	PushStringVec(L, r)
	return 1
}

func LuaIntShowRet(L *lua.State) int {
	L.CheckStack(2)
	tl := GetTasklistFromLua(L)

	tl.luaFlags.showReturnValue = true

	return 0
}

func LuaIntDebulog(L *lua.State) int {
	luaAssertArgnum(L, 1, "Wrong number of arguments to debulog")
	Logf(INFO, "Log from lua: <%s>", L.ToString(1))
	return 0
}

func LuaTableGetString(L *lua.State, key string) string {
	L.PushString(key)
	L.GetTable(-2)
	r := ""
	if !L.IsNil(-1) {
		r = L.ToString(-1)
	}
	L.Pop(1)
	return r
}

func (tl *Tasklist) LuaResultToEntries() ([]*Entry, []string) {
	r := []*Entry{}
	cols := []string{}

	if !tl.luaState.IsTable(-1) {
		panic("Lua function requested to show result but didn't return anything")
	}

	tl.luaState.CheckStack(5)

	tl.luaState.PushString("columns")
	tl.luaState.GetTable(-2)
	if !tl.luaState.IsNil(-1) {
		for j := 1; ; j++ {
			tl.luaState.PushInteger(int64(j))
			tl.luaState.GetTable(-2)
			if tl.luaState.IsNil(-1) {
				tl.luaState.Pop(1)
				break
			}

			cols = append(cols, tl.luaState.ToString(-1))

			tl.luaState.Pop(1)
		}
	}
	tl.luaState.Pop(1)

	fmt.Printf("Columns: %v", cols)

	for j := 1; ; j++ {
		tl.luaState.PushInteger(int64(j))
		tl.luaState.GetTable(-2)
		if tl.luaState.IsNil(-1) {
			tl.luaState.Pop(1)
			break
		}

		id := LuaTableGetString(tl.luaState, "id")
		title := LuaTableGetString(tl.luaState, "title")
		text := LuaTableGetString(tl.luaState, "text")
		sort := LuaTableGetString(tl.luaState, "sort")

		curcols := make(Columns)

		for _, col := range cols {
			tl.luaState.PushString(col)
			tl.luaState.GetTable(-2)
			if !tl.luaState.IsNil(-1) {
				curcols[col] = tl.luaState.ToString(-1)
			}
			tl.luaState.Pop(1)
		}

		e := MakeEntry(id, title, text, NOTES, nil, sort, curcols)

		r = append(r, e)

		tl.luaState.Pop(1)
	}

	tl.luaState.Pop(1)

	return r, cols
}

func (tl *Tasklist) DoStringNoLock(code string, cursor *Entry, freeCursor bool) error {
	if cursor != nil {
		tl.SetEntryInLua(CURSOR, cursor)
	}
	tl.SetTasklistInLua()
	tl.ResetLuaFlags()
	tl.luaFlags.freeCursor = freeCursor
	if tl.executionLimitEnabled {
		tl.luaState.SetExecutionLimit(LUA_EXECUTION_LIMIT)
	}

	if err := tl.luaState.DoString(code); err != nil {
		tl.LogError(fmt.Sprintf("Error while executing lua code: %v", err))
		return err
	}

	return nil
}

func (tl *Tasklist) DoString(code string, cursor *Entry) error {
	return tl.DoStringNoLock(code, cursor, false)
}

func (tl *Tasklist) DoRunString(code string, args []string) error {
	PushStringVec(tl.luaState, args)
	tl.luaState.SetGlobal(RUN_ARGUMENTS_VAR)
	return tl.DoStringNoLock(code, nil, true)
}

func (tl *Tasklist) CallLuaFunction(fname string, cursor *Entry) error {
	tl.SetEntryInLua(CURSOR, cursor)
	tl.SetTasklistInLua()
	tl.ResetLuaFlags()
	if tl.executionLimitEnabled {
		tl.luaState.SetExecutionLimit(LUA_EXECUTION_LIMIT)
	}

	tl.luaState.CheckStack(1)
	tl.luaState.GetGlobal(fname)
	if err := tl.luaState.Call(0, 0); err != nil {
		tl.LogError(fmt.Sprintf("Error while executing lua code: %s", err.Error()))
		return &LuaIntError{err.Error()}
	}

	return nil
}

func NilGlobal(L *lua.State, name string) {
	L.PushNil()
	L.SetGlobal(name)
}

func MakeLuaState() *lua.State {
	L := lua.NewState()
	L.CheckStack(1)

	//L.OpenLibs()
	L.OpenBase()
	L.OpenString()
	L.OpenTable()
	L.OpenMath()
	NilGlobal(L, "collectgarbage")
	NilGlobal(L, "dofile")
	NilGlobal(L, "_G")
	NilGlobal(L, "getfenv")
	NilGlobal(L, "getmetatable")
	NilGlobal(L, "load")
	NilGlobal(L, "loadfile")
	NilGlobal(L, "loadstring")
	NilGlobal(L, "print")
	NilGlobal(L, "rawequal")
	NilGlobal(L, "rawget")
	NilGlobal(L, "rawset")
	NilGlobal(L, "setfenv")
	NilGlobal(L, "setmetatable")
	NilGlobal(L, "coroutine")

	// cursor examination functions

	L.Register("id", LuaIntId)
	L.Register("title", LuaIntTitle)
	L.Register("text", LuaIntText)
	L.Register("priority", LuaIntPriority)
	L.Register("when", LuaIntWhen)
	L.Register("sortfield", LuaIntSortField)

	L.Register("column", LuaIntColumn)
	L.Register("rmcolumn", LuaIntRmColumn)

	// search results control functions

	L.Register("filterout", LuaIntFilterOut)
	L.Register("filterin", LuaIntFilterIn)

	// cursor control functions

	L.Register("visit", LuaIntVisit)

	// editing functions

	L.Register("persist", LuaIntPersist)
	L.Register("remove", LuaIntRemove)
	L.Register("clonecursor", LuaIntCloneCursor)
	L.Register("writecursor", LuaIntWriteCursor)

	// time utility functions

	L.Register("utctime", LuaIntUTCTime)
	L.Register("localtime", LuaIntLocalTime)
	L.Register("timestamp", LuaIntTimestamp)
	L.Register("parsedatetime", LuaIntParseDateTime)

	// string utility functions
	L.Register("split", LuaIntSplit)

	// query construction functions

	L.Register("idq", LuaIntIdQuery)
	L.Register("titleq", LuaIntTitleQuery)
	L.Register("textq", LuaIntTextQuery)
	L.Register("whenq", LuaIntWhenQuery)
	L.Register("searchq", LuaIntSearchQuery)
	L.Register("priorityq", LuaIntPriorityQuery)
	L.Register("columnq", LuaIntColumnQuery)
	L.Register("andq", LuaIntAndQuery)
	L.Register("orq", LuaIntOrQuery)
	L.Register("notq", LuaIntNotQuery)

	// Loads initialization file
	L.DoString(decodeStatic("init.lua"))

	// advanced interface functions
	L.Register("search", LuaIntSearch)
	L.Register("showreturnvalue", LuaIntShowRet)
	//L.Register("debulog", LuaIntDebulog)

	return L
}
