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

func TestTokTime() {
	mmt("#10/2", []string{ "#", "10/2" })
	mmt("#2010-01-21,10:30#l", []string{ "#", "2010-01-21,10:30", "#", "l" })
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
	p := NewParser(t, 0)
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
	tse("#blip?", "blip", "", "")
}


func tae_ex(in string) (*Parser, *AndExpr) {
	t := NewTokenizer(in)
	p := NewParser(t, 0)

	r := &AndExpr{}

	p.ParseAndExpr(r)

	return p, r
}

func check_and_expr(r *AndExpr, expected []string, expVal []string, expExtra []string) {
	if len(r.subExpr) != len(expected) {
		panic(fmt.Sprintf("Different number of returned values found [%v] expected [%v]", r.subExpr, expected))
	}
	
	for i, v := range expected {
		mms(r.subExpr[i].name, v)
		if (expVal != nil) {
			mms(r.subExpr[i].op, "=")
			mms(r.subExpr[i].value, expVal[i])
		}
		if (expExtra != nil) { mms(r.subExpr[i].extra, expExtra[i]) }
	}
}

func tae(in string, expected []string) {
	_, r := tae_ex(in)
	check_and_expr(r, expected, nil, nil)
}

func tae_wval(in string, expected []string, expVal []string, expExtra []string) {
	_, r := tae_ex(in)
	check_and_expr(r, expected, expVal, expExtra)
}

func tae_showcols(in string, expected []string, showCols []string) {
	p, r := tae_ex(in)
	check_and_expr(r, expected, nil, nil)

	if len(p.showCols) != len(showCols) {
		panic(fmt.Sprintf("Different number of renturned values for showCols found [%v] expected [%v]", p.showCols, showCols))
	}

	for i, v := range showCols {
		mms(p.showCols[i], v)
	}
}

func tae_options(in string, expected []string, options []string) {
	p, r := tae_ex(in)
	check_and_expr(r, expected, nil, nil)

	if len(p.options) != len(options) {
		panic(fmt.Sprintf("Different number of options returned [%v] expected [%v]", p.options, options))
	}

	for _, option := range options {
		if _, ok := p.options[option]; !ok {
			panic(fmt.Sprintf("Expected option [%v] not found in [%v]\n", option, p.options))
		}
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
	p := NewParser(t, 0)

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

func TestParsePriority() {
	tae("#l#prova", []string{ "!priority", "prova" })
}

func TestParseTimetag() {
	tentwo_dt, _ := ParseDateTime("10/2", 0)
	tentwo := tentwo_dt.Format(TRIGGER_AT_FORMAT)
	tae_wval("#10/2 prova", []string{ "!when" }, []string{ tentwo }, []string{ "" })
	tae_wval("#10/2+#2010-09-21", []string{ "!when", "!when" }, []string{ tentwo, "2010-09-21 00:00" }, []string{ "", "" })
	tae_wval("#10/2+weekly #2010-09-21", []string{ "!when", "!when" }, []string{ tentwo, "2010-09-21 00:00" }, []string{ "weekly", "" })
}

func TestShowCols() {
	tae_showcols("#blap!>10", []string{ "blap" }, []string{ "blap" })
	tae_showcols("#blap!#blop?#blip", []string{ "blap", "blip" }, []string{ "blap", "blop" })
}

func TestOptions() {
	tae_options("#blap#:w/done", []string{ "blap" }, []string{ "w/done" })
}

func main() {
	fmt.Printf("Testing tokenizer\n")
	TestTokSpaces()
	TestTokMisc()
	TestTokOps()
	TestTokRewind()
	TestTokTime()

	fmt.Printf("Testing parser proper\n")
	TestParseSimpleExpr()
	TestParseAnd()
	TestParseFull()

	fmt.Printf("Testing special tags\n")
	TestParsePriority()
	TestParseTimetag()
	TestShowCols()
	TestOptions()
}
