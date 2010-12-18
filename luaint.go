package main

import (
	"lua51"
	"unsafe"
	"fmt"
)

var CURSOR string = "cursor"
var TASKLIST string = "tasklist"

type LuaFlags struct {
	cursorEdited bool // the original, introduced cursor, was modified
	cursorCloned bool // the cursor was cloned, creating a new entry
}

func (tl *Tasklist) ResetLuaFlags() {
	tl.luaFlags.cursorEdited = false
	tl.luaFlags.cursorCloned = false
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
	
	L.PushString(fmt.Sprintf("Incorrect number of argoments to %s (only 0 or 1 accepted)", fname))
	L.Error()
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
	return LuaIntGetterSetterFunction("triggerat", L,
		func(tl *Tasklist, entry *Entry) string { return entry.TriggerAtString(tl.GetTimezone()) },
		func(tl *Tasklist, entry *Entry, value string) { ta, _ := ParseDateTime(value, tl.GetTimezone()); entry.SetTriggerAt(ta) })
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
	
	L.PushString("Incorrect number of arguments to column (only 1 or 2 accepted)")
	L.Error()
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

func (tl *Tasklist) DoString(code string, cursor *Entry) {
	tl.mutex.Lock()
	defer tl.mutex.Unlock()
	
	tl.SetEntryInLua(CURSOR, cursor)
	tl.SetTasklistInLua()
	tl.ResetLuaFlags()
	
	tl.luaState.DoString(code)
}

func MakeLuaState() *lua51.State {
	L := lua51.NewState()
	L.OpenLibs()

	L.Register("id", LuaIntId)
	L.Register("title", LuaIntTitle)
	L.Register("text", LuaIntText)
	L.Register("priority", LuaIntPriority)
	L.Register("triggerat", LuaIntTriggerAt)
	L.Register("sortfield", LuaIntSortField)
	
	L.Register("column", LuaIntColumn)
	
	L.Register("clonecursor", LuaIntCloneCursor)

	return L
}
