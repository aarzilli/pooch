package main

import (
	"lua51"
	"unsafe"
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
	return *ptr
}

func GetEntryFromLua(L *lua51.State, name string) *Entry {
	L.CheckStack(1)
	L.GetGlobal(name)
	rawptr := L.ToUserdata(-1)
	var ptr **Entry = (**Entry)(rawptr)
	return *ptr
	
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

	L.Register("column", LuaIntColumn)
	L.Register("clonecursor", LuaIntCloneCursor)
	
	return L
}
