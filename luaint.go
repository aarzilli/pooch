package main

import (
	"lua51"
	"unsafe"
	"fmt"
	"time"
	"os"
)

var CURSOR string = "cursor"
var TASKLIST string = "tasklist"
var SEARCHFUNCTION string = "searchfn"

type LuaIntError struct {
	message string
}

func (le *LuaIntError) String() string {
	return le.message
}

type LuaFlags struct {
	cursorEdited bool // the original, introduced cursor, was modified
	cursorCloned bool // the cursor was cloned, creating a new entry
	persist bool // changes are persisted
	filterOut bool // during search filters out the current result
}

func (tl *Tasklist) ResetLuaFlags() {
	tl.luaFlags.cursorEdited = false
	tl.luaFlags.cursorCloned = false
	tl.luaFlags.persist = false
	tl.luaFlags.filterOut = false
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

func LuaError(L *lua51.State, error string) {
	L.CheckStack(1)
	L.PushString(error)
	L.Error()
}

func GetTasklistFromLua(L *lua51.State) *Tasklist {
	L.CheckStack(1)
	L.GetGlobal(TASKLIST)
	rawptr := L.ToUserdata(-1)
	var ptr **Tasklist = (**Tasklist)(rawptr)
	L.Pop(1)
	return *ptr
}

func GetEntryFromLua(L *lua51.State, name string) *Entry {
	L.CheckStack(1)
	L.GetGlobal(name)
	rawptr := L.ToUserdata(-1)
	var ptr **Entry = (**Entry)(rawptr)
	L.Pop(1)
	return *ptr
	
}

func LuaIntGetterSetterFunction(fname string, L *lua51.State, getter func(tl *Tasklist, entry *Entry)string, setter func(tl *Tasklist, entry *Entry, value string)) int {
	argNum := L.GetTop()

	if argNum == 0 {
		entry := GetEntryFromLua(L, CURSOR)
		tl := GetTasklistFromLua(L)
		L.PushString(getter(tl, entry))
		return 1
	} else if argNum == 1 {
		value := L.ToString(1)
		entry := GetEntryFromLua(L, CURSOR)
		tl := GetTasklistFromLua(L)
		setter(tl, entry, value)
		if !tl.luaFlags.cursorCloned { tl.luaFlags.cursorEdited = true }
		return 0
	}
	
	LuaError(L, fmt.Sprintf("Incorrect number of argoments to %s (only 0 or 1 accepted)", fname))
	return 0
}

func LuaIntGetterSetterFunctionInt(fname string, L *lua51.State, getter func(tl *Tasklist, entry *Entry) int, setter func(tl *Tasklist, entry *Entry, value int)) int {
	argNum := L.GetTop()

	if argNum == 0 {
		entry := GetEntryFromLua(L, CURSOR)
		tl := GetTasklistFromLua(L)
		L.PushInteger(getter(tl, entry))
		return 1
	} else if argNum == 1 {
		value := L.ToInteger(1)
		entry := GetEntryFromLua(L, CURSOR)
		tl := GetTasklistFromLua(L)
		setter(tl, entry, value)
		if !tl.luaFlags.cursorCloned { tl.luaFlags.cursorEdited = true }
		return 0
	}
	
	LuaError(L, fmt.Sprintf("Incorrect number of argoments to %s (only 0 or 1 accepted)", fname))
	return 0
}

func LuaIntId(L *lua51.State) int {
	return LuaIntGetterSetterFunction("id", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Id() },
		func(tl *Tasklist, entry *Entry, value string) { if tl.luaFlags.cursorCloned { entry.SetId(value) } })
}

func LuaIntTitle(L *lua51.State) int {
	return LuaIntGetterSetterFunction("title", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Title() },
		func(tl *Tasklist, entry *Entry, value string) { entry.SetTitle(value) })
}

func LuaIntText(L *lua51.State) int {
	return LuaIntGetterSetterFunction("text", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Text() },
		func(tl *Tasklist, entry *Entry, value string) { entry.SetText(value) })
}

func LuaIntSortField(L *lua51.State) int {
	return LuaIntGetterSetterFunction("sortfield", L,
		func(tl *Tasklist, entry *Entry) string { return entry.Sort() },
		func(tl *Tasklist, entry *Entry, value string) { entry.SetSort(value) })
}

func LuaIntPriority(L *lua51.State) int {
	return LuaIntGetterSetterFunction("priority", L,
		func(tl *Tasklist, entry *Entry) string { pr := entry.Priority(); return pr.String() },
		func(tl *Tasklist, entry *Entry, value string) { pr, _ := ParsePriority(value); entry.SetPriority(pr) })
}

func LuaIntTriggerAt(L *lua51.State) int {
	return LuaIntGetterSetterFunctionInt("triggerat", L,
		func(tl *Tasklist, entry *Entry) int { t := entry.TriggerAt(); if t != nil { return int(t.Seconds()) }; return 0 },
		func(tl *Tasklist, entry *Entry, value int) { entry.SetTriggerAt(time.SecondsToUTC(int64(value))) })
}

func LuaIntColumn(L *lua51.State) int {
	argNum := L.GetTop()

	if argNum == 1 {
		name := L.ToString(1)
		entry := GetEntryFromLua(L, CURSOR)
		L.PushString(entry.Column(name))
		return 1
	} else if argNum == 2 {
		name := L.ToString(1)
		value := L.ToString(2)
		entry := GetEntryFromLua(L, CURSOR)
		entry.SetColumn(name, value)
		tl := GetTasklistFromLua(L)
		if !tl.luaFlags.cursorCloned { tl.luaFlags.cursorEdited = true }
		return 0
	}
	
	LuaError(L, "Incorrect number of arguments to column (only 1 or 2 accepted)")
	return 0
}

func LuaIntFilterOut(L *lua51.State) int {
	tl := GetTasklistFromLua(L)
	tl.luaFlags.filterOut = true
	return 0
}

func LuaIntFilterIn(L *lua51.State) int {
	tl := GetTasklistFromLua(L)
	tl.luaFlags.filterOut = false
	return 0
}

func LuaIntPersist(L *lua51.State) int {
	tl := GetTasklistFromLua(L)
	tl.luaFlags.persist = true
	return 0
}

func LuaIntCloneCursor(L *lua51.State) int {
	tl := GetTasklistFromLua(L)
	cursor := GetEntryFromLua(L, CURSOR)
	newcursor := tl.CloneEntry(cursor)
	tl.SetEntryInLua(CURSOR, newcursor)
	tl.luaFlags.cursorCloned = true
	return 0
}

func SetTableInt(L *lua51.State, name string, value int) {
	// Remember to check stack for 2 extra locations
	
	L.PushString(name)
	L.PushInteger(value)
	L.SetTable(-3)
}

func GetTableInt(L *lua51.State, name string) int {
	// Remember to check stack for 1 extra location
	
	L.PushString(name)
	L.GetTable(-2)
	r := L.ToInteger(-1)
	L.Pop(1)
	return r
}

func PushTime(L *lua51.State, t *time.Time) {
	L.CheckStack(3)
	L.CreateTable(0, 7)

	SetTableInt(L, "year", int(t.Year))
	SetTableInt(L, "month", int(t.Month))
	SetTableInt(L, "day", int(t.Day))
	SetTableInt(L, "hour", int(t.Hour))
	SetTableInt(L, "minute", int(t.Minute))
	SetTableInt(L, "second", int(t.Second))
	SetTableInt(L, "offset", int(t.ZoneOffset))
}

func LuaIntUTCTime(L *lua51.State) int {
	if L.GetTop() != 1 {
		LuaError(L, "Wrong number of arguments to utctime")
		return 0
	}
	
	timestamp := L.ToInteger(1)
	PushTime(L, time.SecondsToUTC(int64(timestamp)))
	
	return 1
}

func LuaIntLocalTime(L *lua51.State) int {
	if L.GetTop() != 1 {
		LuaError(L, "Wrong number of arguments to localtime")
		return 0
	}

	tl := GetTasklistFromLua(L)
	timezone := tl.GetTimezone()
	timestamp := L.ToInteger(1)
	
	t := time.SecondsToUTC(int64(timestamp) + (int64(timezone) * 60 * 60))
	t.ZoneOffset = timezone * 60 * 60
	
	PushTime(L, t)
	
	return 1
}

func LuaIntTimestamp(L *lua51.State) int {
	if L.GetTop() != 1 {
		LuaError(L, "Wrong number of arguments to timestamp")
		return 0
	}

	if !L.IsTable(-1) {
		LuaError(L, "Argoment of timestamp is not a table")
		return 0
	}

	L.CheckStack(1)

	var t time.Time

	t.Year = int64(GetTableInt(L, "year"))
	t.Month = GetTableInt(L, "month")
	t.Day = GetTableInt(L, "day")
	t.Hour = GetTableInt(L, "hour")
	t.Minute = GetTableInt(L, "minute")
	t.Second = GetTableInt(L, "second")
	t.ZoneOffset = GetTableInt(L, "offset")

	L.PushInteger(int(t.Seconds()))
	return 1
}

func LuaIntParseDateTime(L *lua51.State) int {
	if L.GetTop() != 1 {
		LuaError(L, "Wrong number of arguments to parsedatetime")
		return 0
	}

	L.CheckStack(1)

	input := L.ToString(-1)
	tl := GetTasklistFromLua(L)

	out, _ := ParseDateTime(input, tl.GetTimezone())

	if out != nil {
		L.PushInteger(int(out.Seconds()))
	} else {
		L.PushInteger(0)
	}

	return 1
}

func (tl *Tasklist) DoString(code string, cursor *Entry) os.Error {
	tl.mutex.Lock()
	defer tl.mutex.Unlock()
	
	tl.SetEntryInLua(CURSOR, cursor)
	tl.SetTasklistInLua()
	tl.ResetLuaFlags()
	
	if !tl.luaState.DoString(code) {
		errorMessage := tl.luaState.ToString(-1)
		tl.LogError(fmt.Sprintf("Error while executing lua code: %s", errorMessage))
		return &LuaIntError{errorMessage}
	}

	return nil
}

func (tl *Tasklist) CallLuaFunction(fname string, cursor *Entry) os.Error {
	tl.mutex.Lock()
	defer tl.mutex.Unlock()

	tl.SetEntryInLua(CURSOR, cursor)
	tl.SetTasklistInLua()
	tl.ResetLuaFlags()

	tl.luaState.CheckStack(1)
	tl.luaState.GetGlobal(fname)
	if lua51.PCall(tl.luaState, 0, 0, 0) != 0 {
		errorMessage := tl.luaState.ToString(-1)
		tl.LogError(fmt.Sprintf("Error while executing lua code: %s", errorMessage))
		return &LuaIntError{errorMessage}
	}

	return nil
}

func MakeLuaState() *lua51.State {
	L := lua51.NewState()
	L.OpenLibs()

	L.CheckStack(1)
	
	L.Register("id", LuaIntId)
	L.Register("title", LuaIntTitle)
	L.Register("text", LuaIntText)
	L.Register("priority", LuaIntPriority)
	L.Register("triggerat", LuaIntTriggerAt)
	L.Register("sortfield", LuaIntSortField)
	
	L.Register("column", LuaIntColumn)

	L.Register("filterout", LuaIntFilterOut)
	L.Register("filterin", LuaIntFilterIn)
	L.Register("persist", LuaIntPersist)
	L.Register("clonecursor", LuaIntCloneCursor)

	L.Register("utctime", LuaIntUTCTime)
	L.Register("localtime", LuaIntLocalTime)
	L.Register("timestamp", LuaIntTimestamp)
	L.Register("parsedatetime", LuaIntParseDateTime)

	return L
}
