package main

import (
	"lua51"
	"fmt"
	"unsafe"
)

func (tl *Tasklist) SetLuaCursor(entry *Entry) {
	fmt.Printf("sizeof entry: %d\n", unsafe.Sizeof(entry))
	rawptr := tl.luaState.NewUserdata(uintptr(unsafe.Sizeof(entry)))
	var ptr **Entry = (**Entry)(rawptr)
	*ptr = entry
	tl.luaState.SetGlobal("cursor")
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
		entry := GetEntryFromLua(L, "cursor")
		L.PushString(entry.Column(name))
		return 1
	} else if argNum == 2 {
		name := L.ToString(1)
		value := L.ToString(2)
		entry := GetEntryFromLua(L, "cursor")
		entry.SetColumn(name, value)
		return 0
	}
	
	L.PushString("Incorrect number of arguments to column (only 1 or 2 accepted)")
	L.Error()
	return 0

	// TODO:
	// - leggere argomenti
}

func MakeLuaState() *lua51.State {
	L := lua51.NewState()
	L.OpenLibs()

	L.Register("column", LuaIntColumn)
		
	return L
}
