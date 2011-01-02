package main

import (
	"fmt"
	"strings"
)

func mms(a string, b string, explanation string) {
	if a != b {
		panic(fmt.Sprintf("Failed matching [%s] to [%s] %s\n", a, b, explanation))
	}
}

func mms_large(a string, b string, explanation string) {
	if a != b {
		panic(fmt.Sprintf("\nFAILED MATCHING:\n\n%s\n\nTO EXPECTED STRING:\n\n%s\n\ncontext: %s\n", a, b, explanation))
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
		mms(x, v, "")
	}
	mms(t.Next(), "", "")
}

func TestTokSpaces() {
	mmt("  prova", []string{ " ", "prova" })
	mmt("prova    ", []string{ "prova", " " })
}

func TestTokMisc() {
	mmt("#prova^^^bau", []string{ "#", "prova", "^^^bau" })
	mmt("#prova +#prova", []string{ "#", "prova", " ", "+", "#", "prova" })
	mmt("#blip#blop", []string{ "#", "blip", "#", "blop" })
	mmt("#prova#prova+#prova@prova", []string{ "#", "prova", "#", "prova+", "#", "prova", "#", "prova" })
}

func TestTokTime() {
	mmt("#10/2", []string{ "#", "10/2" })
	mmt("#2010-01-21,10:30#l", []string{ "#", "2010-01-21,10:30", "#", "l" })
}

func TestTokOps() {
	mmt(" = <a <= ! >", []string{
		" ", "=",
		" ", "<", "a",
		" ", "<=",
		" ", "!",
		" ", ">", 
	})
}

func TestTokRewind() {
	t := NewTokenizer("bli bla bolp blap")
	
	mms(t.Next(), "bli", "")
	mms(t.Next(), " ", "")
	
	pos := t.next
	
	mms(t.Next(), "bla", "")
	mms(t.Next(), " ", "")
	
	t.next = pos

	mms(t.Next(), "bla", "")
	mms(t.Next(), " ", "")
	mms(t.Next(), "bolp", "")
	mms(t.Next(), " ", "")
	mms(t.Next(), "blap", "")
	mms(t.Next(), "", "")

	t.next = pos

	mms(t.Next(), "bla", "")
	mms(t.Next(), " ", "")
	mms(t.Next(), "bolp", "")
	mms(t.Next(), " ", "")
	mms(t.Next(), "blap", "")
	mms(t.Next(), "", "")
}

func tse(in string, name string, op string, value string) {
	t := NewTokenizer(in)
	p := NewParser(t, 0)
	expr := &SimpleExpr{}
	
	if !p.ParseSimpleExpression(expr) {
		panic("Couldn't parse expression: [" + in + "]")
	}

	mms(expr.name, name, " matching name")
	mms(expr.op, op, " matching operation")
	mms(expr.value, value, " matching value")
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


func tae_ex(in string) (*Parser, *ParseResult) {
	t := NewTokenizer(in)
	p := NewParser(t, 0)

	r := p.ParseEx()

	return p, r
}

func check_and_expr(r *AndExpr, expected []string, expVal []string, expExtra []string) {
	if len(r.subExpr) != len(expected) {
		panic(fmt.Sprintf("Different number of returned values found [%v] expected [%v]", r.subExpr, expected))
	}
	
	for i, v := range expected {
		mms(r.subExpr[i].name, v, "matching name")
		if (expVal != nil) {
			mms(r.subExpr[i].op, "=", "matching operator")
			mms(r.subExpr[i].value, expVal[i], "matching value")
		}
		if (expExtra != nil) { mms(r.subExpr[i].extra, expExtra[i], "matching extra content") }
	}
}

func tae(in string, expected []string) {
	_, r := tae_ex(in)
	check_and_expr(&(r.include), expected, nil, nil)
}

func tae_wval(in string, expected []string, expVal []string, expExtra []string) {
	_, r := tae_ex(in)
	check_and_expr(&(r.include), expected, expVal, expExtra)
}

func tae_showcols(in string, expected []string, showCols []string) {
	p, r := tae_ex(in)
	check_and_expr(&(r.include), expected, nil, nil)

	if len(p.showCols) != len(showCols) {
		panic(fmt.Sprintf("Different number of renturned values for showCols found [%v] expected [%v]", p.showCols, showCols))
	}

	for i, v := range showCols {
		mms(p.showCols[i], v, "matching shown columns")
	}
}

func tae_options(in string, expected []string, options []string) {
	p, r := tae_ex(in)
	check_and_expr(&(r.include), expected, nil, nil)

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

func tae2(in string, includeExpected []string, excludeExpected []string, query string) {
	t := NewTokenizer(in)
	p := NewParser(t, 0)

	r := p.ParseEx()

	if len(r.include.subExpr) != len(includeExpected) {
		panic(fmt.Sprintf("Different number of ored terms [%v] expected [%v]", r.include, includeExpected))
	}

	for i, v := range includeExpected {
		mms(r.include.subExpr[i].name, v, "matching name of normal expression")
	}

	if len(r.exclude.subExpr) != len(excludeExpected) {
		panic(fmt.Sprintf("Different number of removed [%v] expected [%v]", r.exclude, excludeExpected))
	}

	for i, v := range excludeExpected {
		mms(r.exclude.subExpr[i].name, v, "matching name of excluded expression")
	}

	mms(strings.Trim(r.text, " "), strings.Trim(query, " "), "matching query")
}

func TestParseFull() {
	tae2("blip #blip#blop",
		[]string{ "blip", "blop" },
		[]string{},
		"blip")
	
	tae2("blip #blip#blop blap",
		[]string{ "blip", "blop" },
		[]string{},
		"blip blap")
	
	tae2("blip #blip#blop blap -#balp",
		[]string{ "blip", "blop" },
		[]string{ "balp" },
		"blip blap")
}

func TestParsePriority() {
	tae("#l#prova", []string{ ":priority", "prova" })
}

func TestParseTimetag() {
	tentwo_dt, _ := ParseDateTime("10/2", 0)
	tentwo := tentwo_dt.Format(TRIGGER_AT_FORMAT)
	tae_wval("#10/2 prova", []string{ ":when" }, []string{ tentwo }, []string{ "" })
	tae_wval("#10/2 #2010-09-21", []string{ ":when", ":when" }, []string{ tentwo, "2010-09-21 00:00" }, []string{ "", "" })
	tae_wval("#10/2+weekly #2010-09-21", []string{ ":when", ":when" }, []string{ tentwo, "2010-09-21 00:00" }, []string{ "weekly", "" })
}

func TestShowCols() {
	tae_showcols("#blap!>10", []string{ "blap" }, []string{ "blap" })
	tae_showcols("#blap!#blop?#blip", []string{ "blap", "blip" }, []string{ "blap", "blop" })
}

func TestOptions() {
	tae_options("#blap#:w/done", []string{ "blap" }, []string{ "w/done" })
}

func TestSavedSearch() {
	p, r := tae_ex("#%salvata")
	check_and_expr(&(r.include), []string{ }, nil, nil)
	mms(p.savedSearch, "salvata", "")
}

func TestEscaping() {
	_, r  := tae_ex("blip @@ blap ## blop")
	mms(r.text, "blip @ blap # blop", "")
}

func textra(input string, normal string, extra string, command string) {
	t := NewTokenizer(input)
	p := NewParser(t, 0)

	r := p.ParseEx()

	mms(r.text, normal, "")
	mms(p.extra, extra, "")
	mms(p.command, command, "")
}

func TestExtra() {
	textra("prova bi #blap#+ questo e` tutto extra", "prova bi", " questo e` tutto extra", "")
	textra("prova bi #blap#+", "prova bi", "", "")
	textra("prova bi #blap#+ questo e` tutto extra #! questo e` un comando", "prova bi", " questo e` tutto extra ", " questo e` un comando")
	textra("prova bi #+ questo e` tutto extra #", "prova bi", " questo e` tutto extra #", "")
}

func mme(a, b *Entry) {
	if b.id != "" { mms(a.id, b.id, "matching entry id") }
	mms(a.title, b.title, "matching entry title")
	mms(a.text, b.text, "matching entry text")
	if a.priority != b.priority {
		panic(fmt.Sprintf("Cannot match priority %d to %d\n", a.priority, b.priority))
	}
	if (a.triggerAt != nil) != (b.triggerAt != nil) {
		panic(fmt.Sprintf("Different values for triggerAt (nilness: %v %v)\n", (a.triggerAt == nil), (b.triggerAt == nil)))
	}
	if a.triggerAt != nil {
		mms(a.triggerAt.Format(TRIGGER_AT_FORMAT), b.triggerAt.Format(TRIGGER_AT_FORMAT), "matching triggerAt")
	}
	if b.sort != "" { mms(a.sort, b.sort, "matching entry sort") }
	if len(a.columns) != len(b.columns) {
		panic(fmt.Sprintf("Different number of columns %v and %v\n", a.columns, b.columns))
	}
	for k, v := range a.columns {
		v2, ok := b.columns[k]
		if !ok {
			panic(fmt.Sprintf("Column mismatch on key %s (missing) on %v and %v\n", k, a.columns, b.columns))
		}
		mms(v, v2, fmt.Sprintf("value mismatch on key %s on %v and %v", k, a.columns, b.columns))
	}
}

func tpn(tl *Tasklist, entryText string, queryText string, entry *Entry) {
	mme(tl.ParseNew(entryText, queryText), entry)
}

func TestSimpleEntry(tl *Tasklist) {
	tpn(tl, "prova prova @blap", "",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": ""}))
	
	tpn(tl, "prova prova @blip = blop anta", "",
		MakeEntry("", "prova prova anta", "", NOW, nil, "",
		map[string]string{"uncat": "", "blip": "blop"}))

}

func TestColEntry(tl *Tasklist) {
	tpn(tl, "prova prova #+\nblip: blop\nblap:\n", "",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": "", "blip": "blop"}))

	tpn(tl, "prova prova #+\nblip: blop\n", "",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"uncat": "", "blip": "blop"}))
}

func TestSpecialEntry(tl *Tasklist) {
	tpn(tl, "prova prova #id=ciao", "",
		MakeEntry("ciao", "prova prova", "", NOW, nil, "",
		map[string]string{"uncat": ""}))

	tpn(tl, "#l prova prova", "",
		MakeEntry("", "prova prova", "", LATER, nil, "",
		map[string]string{"uncat": ""}))

	tpn(tl, "#blap#l prova prova", "",
		MakeEntry("", "prova prova", "", LATER, nil, "",
		map[string]string{"blap": ""}))

	t, _ := ParseDateTime("2010-10-01", 0)
	tpn(tl, "#2010-10-01 #blap prova prova", "",
		MakeEntry("", "prova prova", "", TIMED, t, "",
		map[string]string{"blap": ""}))
}

func TestEntryWithSearch(tl *Tasklist) {
	tpn(tl, "prova prova", "prova #blap",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": ""}))
	
	tpn(tl, "prova prova", "prova #blap#blop",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": "", "blop": ""}))
}

func tis(tl *Tasklist, input string, expectedOutput string) {
	output, _, err := tl.ParseSearch(input)
	must(err)
	mms_large(output, "SELECT tasks.id, title_field, text_field, priority, trigger_at_field, sort, group_concat(columns.name||':'||columns.value, '\v')\nFROM tasks NATURAL JOIN columns" + expectedOutput + "\nGROUP BY tasks.id\nORDER BY priority, trigger_at_field ASC, sort DESC", "")
	stmt, err := tl.conn.Prepare("EXPLAIN " + output)
	must(err)
	defer stmt.Finalize()
	must(stmt.Exec())
}

func TestNoQuerySelect(tl *Tasklist) {
	tis(tl, "", "\nWHERE\n   priority <> 5")
	
	tis(tl, "#bib", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   priority <> 5")
	
	tis(tl, "#l", "\nWHERE\n   priority = 2")
	tis(tl, "#2010-10-2", "\nWHERE\n   trigger_at_field = '2010-10-02 00:00'\nAND\n   priority <> 5")
	
	tis(tl, "#bib#l", "\nWHERE\n   priority = 2\nAND\n   id IN (SELECT id FROM columns WHERE name = 'bib')")

	tis(tl, "#bib#bab#bob", "\nWHERE\n   id IN (\n      SELECT id FROM columns WHERE name = 'bib'\n   INTERSECT\n      SELECT id FROM columns WHERE name = 'bab'\n   INTERSECT\n      SELECT id FROM columns WHERE name = 'bob')\nAND\n   priority <> 5")

	tis(tl, "#bib#bab#2010-10-02", "\nWHERE\n   trigger_at_field = '2010-10-02 00:00'\nAND\n   id IN (\n      SELECT id FROM columns WHERE name = 'bib'\n   INTERSECT\n      SELECT id FROM columns WHERE name = 'bab')\nAND\n   priority <> 5")
}

func TestExclusionSelect(tl *Tasklist) {
	tis(tl, "-#bib", "\nWHERE\n   priority <> 5\nAND\n   id NOT IN (SELECT id FROM columns WHERE name = 'bib')")

	tis(tl, "#bib -#bab", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   priority <> 5\nAND\n   id NOT IN (SELECT id FROM columns WHERE name = 'bab')")

	tis(tl, "#bib -#bab -#bob", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   priority <> 5\nAND\n   id NOT IN (\n      SELECT id FROM columns WHERE name = 'bab'\n   INTERSECT\n      SELECT id FROM columns WHERE name = 'bob')")
}

func TestOptionsSelect(tl *Tasklist) {
	tis(tl, "#bib#:w/done", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')")
}

func TestSavedSearchSelect(tl *Tasklist) {
	tis(tl, "#%idontexist", "\nWHERE\n   priority <> 5")
}

func TestQuerySelect(tl *Tasklist) {
	tis(tl, "prova prova #bla#blo", "\nWHERE\n   id IN (\n      SELECT id FROM columns WHERE name = 'bla'\n   INTERSECT\n      SELECT id FROM columns WHERE name = 'blo')\nAND\n   priority <> 5\nAND\n   id IN (\n      SELECT id FROM ridx WHERE title_field MATCH 'prova prova'\n   UNION\n      SELECT id FROM ridx WHERE text_field MATCH 'prova prova')")
}

func SetupSearchStuff(tl *Tasklist) {
	tl.Add(tl.ParseNew("#id=10#bla questa è una prova #blo", ""))
	tl.Add(tl.ParseNew("#id=11#bib=10 ging bong un #bla", ""))
	tl.Add(tl.ParseNew("#id=12#bib=20#bla questa è una prova", ""))
	tl.Add(tl.ParseNew("#id=13#2010-01-01 bang", ""))
	tl.Add(tl.ParseNew("#id=14#2010-10-10 bang", ""))
}

func tsearch(tl *Tasklist, queryText string, expectedIds []string) {
	ids := make(map[string]string)

	for _, id := range expectedIds { ids[id] = "" }

	theselect, code, err := tl.ParseSearch(queryText)
	must(err)
	entries, err := tl.Retrieve(theselect, code)
	must(err)

	if len(entries) != len(ids) {
		panic(fmt.Sprintf("Wrong number of entries in result %d (expected %d)", len(entries), len(ids)))
	}
	
	for _, entry := range entries {
		if _, ok := ids[entry.Id()]; !ok {
			panic(fmt.Sprintf("Unexpected id %s in result", entry.Id()))
		}
	}
}

func TestSearch(tl *Tasklist) {
	tsearch(tl, "#bla", []string{ "10", "11", "12" })
	tsearch(tl, "#bib=10", []string{ "11" })
	tsearch(tl, "prova", []string{ "10", "12" })
	tsearch(tl, "prova #blo", []string{ "10" })
}

func TestLuaSelect(tl *Tasklist) {
	tsearch(tl, "prova #+ idq('10')", []string{ "10" })
	tsearch(tl, "borva #+ idq('10')", []string{ })
	tsearch(tl, "#+ idq('10')", []string{ "10" })
	tsearch(tl, "#+ titleq('=', 'ging bong un')", []string{ "11" })

	tsearch(tl, "bang", []string{ "13", "14" })
	tsearch(tl, "bang #+ whenq('>', 1275775200)", []string{ "14" })
	tsearch(tl, "bang #+ whenq('<', 1275775200)", []string{ "13" })

	tsearch(tl, "#+ titleq('match', 'prova')", []string{ "10", "12" })
	tsearch(tl, "#+ searchq('prova')", []string{ "10", "12" })
	
	theselect, _, err := tl.ParseSearch("prova #+ whenq('>', 1275775200)")
	must(err)
	fmt.Printf("%s \n", theselect)
}

func main() {
	tl := OpenOrCreate("/tmp/testing.pooch")
	defer tl.Close()
	tl.Truncate()
	SetupSearchStuff(tl)

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
	TestSavedSearch()
	TestEscaping()
	TestExtra()

	fmt.Printf("Testing new entry creation\n")
	TestSimpleEntry(tl)
	TestColEntry(tl)
	TestSpecialEntry(tl)
	TestEntryWithSearch(tl)

	fmt.Printf("Testing compile into select\n")
	TestNoQuerySelect(tl)
	TestExclusionSelect(tl)
	TestOptionsSelect(tl)
	TestSavedSearchSelect(tl)
	TestQuerySelect(tl)

	fmt.Printf("Testing actual search into backend\n")
	TestSearch(tl)

	fmt.Printf("Testing lua interface (for query creation)\n")
	TestLuaSelect(tl)
}
