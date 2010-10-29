package main

import (
	"fmt"
	"container/vector"
)

func mms(a string, b string) {
	if a != b {
		panic(fmt.Sprintf("Failed matching [%s] to [%s]\n", a, b))
	}
}

func mmt(a string, b []string) {
	var uptohere vector.StringVector
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Failed matching [%v] to [%v]\n", []string(uptohere), b))
		}
	}()
	t := NewTokenizer(a)
	for _, v := range b {
		x := t.Next()
		uptohere.Push(x)
		mms(x, v)
	}
	mms(t.Next(), "")
}

func TestTokSpaces() {
	mmt("  prova", []string{ " ", "prova" })
	mmt("prova    ", []string{ "prova", " " })
}

func TestTokMisc() {
	mmt("#prova^^^bau", []string{ "#", "prova", "^^^bau" })
	mmt("#prova +#prova", []string{ "#", "prova", " ", "+", "#", "prova" })
	mmt("#prova#prova+#prova@prova", []string{ "#", "prova", "#", "prova", "+", "#", "prova", "#", "prova" })
}

func TestTokOps() {
	mmt(" = =~ <a <= !~ ! >", []string{
		" ", "=",
		" ", "=~",
		" ", "<", "a",
		" ", "<=",
		" ", "!~",
		" ", "!",
		" ", ">", 
	})
}

func TestTokRewind() {
	t := NewTokenizer("bli bla bolp blap")
	
	mms(t.Next(), "bli")
	mms(t.Next(), " ")
	
	pos := t.next
	
	mms(t.Next(), "bla")
	mms(t.Next(), " ")
	
	t.next = pos

	mms(t.Next(), "bla")
	mms(t.Next(), " ")
	mms(t.Next(), "bolp")
	mms(t.Next(), " ")
	mms(t.Next(), "blap")
	mms(t.Next(), "")

	t.next = pos

	mms(t.Next(), "bla")
	mms(t.Next(), " ")
	mms(t.Next(), "bolp")
	mms(t.Next(), " ")
	mms(t.Next(), "blap")
	mms(t.Next(), "")
}

func tse(in string, name string, op string, value string) {
	t := NewTokenizer(in)
	p := NewParser(t)
	expr := &SimpleExpr{}
	
	if !p.ParseSimpleExpression(&expr) {
		panic("Couldn't parse expression: [" + in + "]")
	}

	mms(expr.name, name)
	mms(expr.op, op)
	mms(expr.value, value)
}

func TestParseSimpleExpr() {
	tse("#blip", "blip", "", "")
	tse("#blip!", "blip", "", "")
	tse("#blip! > 0", "blip", ">", "0")
	tse("#blip > 0", "blip", ">", "0")
	tse("#blip>0", "blip", ">", "0")
	tse("#blip!>0", "blip", ">", "0")
}

func main() {
	TestTokSpaces()
	TestTokMisc()
	TestTokOps()
	TestTokRewind()
	TestParseSimpleExpr()
}