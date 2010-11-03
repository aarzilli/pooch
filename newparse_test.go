package main

import (
	"fmt"
	"strings"
)

func mms(a string, b string) {
	if a != b {
		panic(fmt.Sprintf("Failed matching [%s] to [%s]\n", a, b))
	}
}

func mmt(a string, b []string) {
	uptohere := make([]string, 0)
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Sprintf("Failed matching [%v] to [%v]\n", []string(uptohere), b))
		}
	}()
	t := NewTokenizer(a)
	for _, v := range b {
		x := t.Next()
		uptohere = append(uptohere, x)
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
	mmt("#blip#blop", []string{ "#", "blip", "#", "blop" })
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
	
	if !p.ParseSimpleExpression(expr) {
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

func tae(in string, expected []string) {
	t := NewTokenizer(in)
	p := NewParser(t)

	r := &AndExpr{}

	p.ParseAndExpr(r)

	if len(r.subExpr) != len(expected) {
		panic(fmt.Sprintf("Different number of returned values found [%v] expected [%v]", r.subExpr, expected))
	}

	for i, v := range expected {
		mms(r.subExpr[i].name, v)
	}
}

func TestParseAnd() {
	tae("#blip", []string{ "blip" })
	tae("#blip #blop", []string{ "blip", "blop" })
	tae("#blip#blop", []string{ "blip", "blop" })
	tae("#blip > 20 #blop", []string{ "blip", "blop" })
	tae("#blip>20#blop", []string{ "blip", "blop" })
}

func tae2(in string, oredExpected [][]string, removedExpected []string, query string) {
	t := NewTokenizer(in)
	p := NewParser(t)

	r := p.Parse()

	if len(r.ored) != len(oredExpected) {
		panic(fmt.Sprintf("Different number of ored terms [%v] expected [%v]", r.ored, oredExpected))
	}

	for i, v := range oredExpected {
		if len(v) != len(r.ored[i].subExpr) {
			panic(fmt.Sprintf("Different number of ored subterms [%v] expected [%v]", r.ored[i].subExpr, v))
		}
		
		for j, w := range v {
			mms(r.ored[i].subExpr[j].name, w)
		}
	}

	if len(r.removed) != len(removedExpected) {
		panic(fmt.Sprintf("Different number of removed [%v] expected [%v]", r.removed, removedExpected))
	}

	for i, v := range removedExpected {
		mms(r.removed[i].name, v)
	}

	mms(strings.Trim(r.query, " "), strings.Trim(query, " "))
}

func TestParseFull() {
	tae2("blip #blip#blop",
		[][]string{ []string{ "blip", "blop" } },
		[]string{},
		"blip")
	tae2("blip #blip#blop blap",
		[][]string{ []string{ "blip", "blop" } },
		[]string{},
		"blip blap")
	tae2("blip #blip#blop blap -#balp",
		[][]string{ []string{ "blip", "blop" } },
		[]string{ "balp" },
		"blip blap")
	tae2("#blip#blap +#blop#blup",
		[][]string{ []string{ "blip", "blap" }, []string{ "blop", "blup" } },
		[]string{},
		"")
}

func main() {
	TestTokSpaces()
	TestTokMisc()
	TestTokOps()
	TestTokRewind()
	TestParseSimpleExpr()
	TestParseAnd()
	TestParseFull()
}