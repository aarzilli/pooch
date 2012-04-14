/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package pooch

import (
	"fmt"
	"strings"
	"testing"
)

func mms(z *testing.T, a string, b string, explanation string) {
	if a != b {
		z.Errorf("Failed matching [%s] to [%s] %s\n", a, b, explanation)
	}
}

func mms_large(z *testing.T, a string, b string, explanation string) {
	if a != b {
		z.Errorf("\nFAILED MATCHING:\n\n%s\n\nTO EXPECTED STRING:\n\n%s\n\ncontext: %s\n", a, b, explanation)
	}
}

func mmt(z *testing.T, a string, b []string) {
	uptohere := make([]string, 0)
	defer func() {
		if r := recover(); r != nil {
			z.Errorf("Failed matching [%v] to [%v]\n", []string(uptohere), b)
		}
	}()
	t := NewTokenizer(a)
	for _, v := range b {
		x := t.Next()
		uptohere = append(uptohere, x)
		mms(z, x, v, "")
	}
	mms(z, t.Next(), "", "")
}

func TestTokSpaces(z *testing.T) {
	fmt.Println("TestTokSpaces")
	mmt(z, "  prova", []string{ " ", "prova" })
	mmt(z, "prova    ", []string{ "prova", " " })
}

func TestTokMisc(z *testing.T) {
	fmt.Println("TestTokMisc")
	mmt(z, "#prova^^^bau", []string{ "#", "prova", "^^^bau" })
	mmt(z, "#prova +#prova", []string{ "#", "prova", " ", "+", "#", "prova" })
	mmt(z, "#blip#blop", []string{ "#", "blip", "#", "blop" })
	mmt(z, "#prova#prova+#prova@prova", []string{ "#", "prova", "#", "prova+", "#", "prova", "#", "prova" })
}

func TestTokTime(z *testing.T) {
	fmt.Println("TestTokTime")
	mmt(z, "#10/2", []string{ "#", "10/2" })
	mmt(z, "#2010-01-21,10:30#l", []string{ "#", "2010-01-21,10:30", "#", "l" })
}

func TestTokOps(z *testing.T) {
	fmt.Println("TestTokOps")
	mmt(z, " = <a <= ! >", []string{
		" ", "=",
		" ", "<", "a",
		" ", "<=",
		" ", "!",
		" ", ">",
	})
}

func TestTokRewind(z *testing.T) {
	fmt.Println("TestTokRewind")
	t := NewTokenizer("bli bla bolp blap")

	mms(z, t.Next(), "bli", "")
	mms(z, t.Next(), " ", "")

	pos := t.next

	mms(z, t.Next(), "bla", "")
	mms(z, t.Next(), " ", "")

	t.next = pos

	mms(z, t.Next(), "bla", "")
	mms(z, t.Next(), " ", "")
	mms(z, t.Next(), "bolp", "")
	mms(z, t.Next(), " ", "")
	mms(z, t.Next(), "blap", "")
	mms(z, t.Next(), "", "")

	t.next = pos

	mms(z, t.Next(), "bla", "")
	mms(z, t.Next(), " ", "")
	mms(z, t.Next(), "bolp", "")
	mms(z, t.Next(), " ", "")
	mms(z, t.Next(), "blap", "")
	mms(z, t.Next(), "", "")
}

func tse(z *testing.T, in string, name string, op string, value string) {
	t := NewTokenizer(in)
	p := NewParser(t, 0)
	expr := &SimpleExpr{}

	if !p.ParseSimpleExpression(expr) {
		z.Error("Couldn't parse expression: [" + in + "]")
	}

	mms(z, expr.name, name, " matching name")
	mms(z, expr.op, op, " matching operation")
	mms(z, expr.value, value, " matching value")
}

func TestParseSimpleExpr(z *testing.T) {
	tse(z, "#blip", "blip", "", "")
	tse(z, "#blip!", "blip", "", "")
	tse(z, "#blip! > 0", "blip", ">", "0")
	tse(z, "#blip > 0", "blip", ">", "0")
	tse(z, "#blip>0", "blip", ">", "0")
	tse(z, "#blip!>0", "blip", ">", "0")
	tse(z, "#blip?", "blip", "", "")
}


func tae_ex(in string) (*Parser, *ParseResult) {
	t := NewTokenizer(in)
	p := NewParser(t, 0)

	r := p.ParseEx()

	return p, r
}

func check_and_expr(z *testing.T, r *BoolExpr, expected []string, expVal []string, expExtra []string) {
	if len(r.subExpr) != len(expected) {
		z.Errorf("Different number of returned values found [%v] expected [%v]", r.subExpr, expected)
	}

	for i, v := range expected {
		sexpr := r.subExpr[i].(*SimpleExpr)
		if sexpr == nil {
			z.Error("Found a subexpression that isn't simple")
		}
		mms(z, sexpr.name, v, "matching name")
		if (expVal != nil) {
			mms(z, sexpr.op, "=", "matching operator")
			mms(z, sexpr.value, expVal[i], "matching value")
		}
		if (expExtra != nil) { mms(z, sexpr.extra, expExtra[i], "matching extra content") }
	}
}

func tae(z *testing.T, in string, expected []string) {
	_, r := tae_ex(in)
	check_and_expr(z, &(r.include), expected, nil, nil)
}

func tae_wval(z *testing.T, in string, expected []string, expVal []string, expExtra []string) {
	_, r := tae_ex(in)
	check_and_expr(z, &(r.include), expected, expVal, expExtra)
}

func tae_showcols(z *testing.T, in string, expected []string, showCols []string) {
	_, r := tae_ex(in)
	check_and_expr(z, &(r.include), expected, nil, nil)

	if len(r.showCols) != len(showCols) {
		z.Errorf("Different number of renturned values for showCols found [%v] expected [%v]", r.showCols, showCols)
	}

	for i, v := range showCols {
		mms(z, r.showCols[i], v, "matching shown columns")
	}
}

func tae_options(z *testing.T, in string, expected []string, options []string) {
	_, r := tae_ex(in)
	check_and_expr(z, &(r.include), expected, nil, nil)

	if len(r.options) != len(options) {
		z.Errorf("Different number of options returned [%v] expected [%v]", r.options, options)
	}

	for _, option := range options {
		if _, ok := r.options[option]; !ok {
			z.Errorf("Expected option [%v] not found in [%v]\n", option, r.options)
		}
	}
}

func TestParseAnd(z *testing.T) {
	fmt.Println("TestParseAnd")
	tae(z, "#blip", []string{ "blip" })
	tae(z, "#blip #blop", []string{ "blip", "blop" })
	tae(z, "#blip#blop", []string{ "blip", "blop" })
	tae(z, "#blip > 20 #blop", []string{ "blip", "blop" })
	tae(z, "#blip>20#blop", []string{ "blip", "blop" })
}

func tae2(z *testing.T, in string, includeExpected []string, excludeExpected []string, query string) {
	t := NewTokenizer(in)
	p := NewParser(t, 0)

	r := p.ParseEx()

	if len(r.include.subExpr) != len(includeExpected) {
		z.Errorf("Different number of ored terms [%v] expected [%v]", r.include, includeExpected)
	}

	for i, v := range includeExpected {
		sexpr := r.include.subExpr[i].(*SimpleExpr)
		if sexpr == nil {
			z.Error("Non-simple subexpression found")
		}
		mms(z, sexpr.name, v, "matching name of normal expression")
	}

	if len(r.exclude.subExpr) != len(excludeExpected) {
		z.Errorf("Different number of removed [%v] expected [%v]", r.exclude, excludeExpected)
	}

	for i, v := range excludeExpected {
		sexpr := r.exclude.subExpr[i].(*SimpleExpr)
		if sexpr == nil {
			z.Error("Non-simple subexpression found")
		}
		mms(z, sexpr.name, v, "matching name of excluded expression")
	}

	mms(z, strings.Trim(r.text, " "), strings.Trim(query, " "), "matching query")
}

func TestParseFull(z *testing.T) {
	fmt.Println("TestParseFull")
	tae2(z, "blip #blip#blop",
		[]string{ "blip", "blop" },
		[]string{},
		"blip")

	tae2(z, "blip #blip#blop blap",
		[]string{ "blip", "blop" },
		[]string{},
		"blip blap")

	tae2(z, "blip #blip#blop blap -#balp",
		[]string{ "blip", "blop" },
		[]string{ "balp" },
		"blip blap")
}

func TestParsePriority(z *testing.T) {
	fmt.Println("TestParsePriority")
	tae(z, "#l#prova", []string{ ":priority", "prova" })
}

func TestParseTimetag(z *testing.T) {
	fmt.Println("TestParseTimetag")
	tentwo_dt, _ := ParseDateTime("10/2", 0)
	tentwo := tentwo_dt.Format(TRIGGER_AT_FORMAT)
	tae_wval(z, "#10/2 prova", []string{ ":when" }, []string{ tentwo }, []string{ "" })
	tae_wval(z, "#10/2 #2010-09-21", []string{ ":when", ":when" }, []string{ tentwo, "2010-09-21 00:00" }, []string{ "", "" })
	tae_wval(z, "#10/2+weekly #2010-09-21", []string{ ":when", ":when" }, []string{ tentwo, "2010-09-21 00:00" }, []string{ "weekly", "" })

	datetime, err := ParseDateTime("13:40", 0)
	Must(err)
	if (datetime.Hour() != 13) || (datetime.Minute() != 40) || (datetime.Year() < 2012) {
		z.Error("Error parsing hour only time expression")
	}

	datetime, err = ParseDateTime("Thu", 0)
	Must(err)
	if (datetime.Year() < 2010) || (datetime.Weekday() != 4) {
		z.Errorf("Error parsing day of the week time expression: %s, weekday: %d", datetime.Format(TRIGGER_AT_FORMAT), datetime.Weekday())
	}
	fmt.Printf("Parsed: %s\n", datetime.Format(TRIGGER_AT_FORMAT))
}

func TestShowCols(z *testing.T) {
	fmt.Println("TestShowCols")
	tae_showcols(z, "#blap!>10", []string{ "blap" }, []string{ "blap" })
	tae_showcols(z, "#blap!#blop?#blip", []string{ "blap", "blip" }, []string{ "blap", "blop" })
}

func TestOptions(z *testing.T) {
	fmt.Println("TestOptions")
	tae_options(z, "#blap#:w/done", []string{ "blap" }, []string{ "w/done" })
}

func TestSavedSearch(z *testing.T) {
	fmt.Println("TestSavedSearch")
	_, r := tae_ex("#%salvata")
	check_and_expr(z, &(r.include), []string{ }, nil, nil)
	mms(z, r.savedSearch, "salvata", "")
}

func TestEscaping(z *testing.T) {
	fmt.Println("TestEscaping")
	_, r  := tae_ex("blip @@ blap ## blop")
	mms(z, r.text, "blip @ blap # blop", "")
}

func textra(z *testing.T, input string, normal string, extra string, command string) {
	t := NewTokenizer(input)
	p := NewParser(t, 0)

	r := p.ParseEx()

	mms(z, r.text, normal, "")
	mms(z, r.extra, extra, "")
	mms(z, r.command, command, "")
}

func TestExtra(z *testing.T) {
	textra(z, "prova bi #blap#+ questo e` tutto extra", "prova bi", " questo e` tutto extra", "")
	textra(z, "prova bi #blap#+", "prova bi", "", "")
	textra(z, "prova bi #blap#+ questo e` tutto extra #! questo e` un comando", "prova bi", " questo e` tutto extra ", " questo e` un comando")
	textra(z, "prova bi #+ questo e` tutto extra #", "prova bi", " questo e` tutto extra #", "")
}

func mme(z *testing.T, a, b *Entry) {
	if b.id != "" { mms(z, a.id, b.id, "matching entry id") }
	mms(z, a.title, b.title, "matching entry title")
	mms(z, a.text, b.text, "matching entry text")
	if a.priority != b.priority {
		z.Errorf("Cannot match priority %d to %d\n", a.priority, b.priority)
	}
	if (a.triggerAt != nil) != (b.triggerAt != nil) {
		z.Errorf("Different values for triggerAt (nilness: %v %v)\n", (a.triggerAt == nil), (b.triggerAt == nil))
	}
	if a.triggerAt != nil {
		mms(z, a.triggerAt.Format(TRIGGER_AT_FORMAT), b.triggerAt.Format(TRIGGER_AT_FORMAT), "matching triggerAt")
	}
	if b.sort != "" { mms(z, a.sort, b.sort, "matching entry sort") }
	if len(a.columns) != len(b.columns) {
		z.Errorf("Different number of columns %v and %v\n", a.columns, b.columns)
	}
	for k, v := range a.columns {
		v2, ok := b.columns[k]
		if !ok {
			z.Errorf("Column mismatch on key %s (missing) on %v and %v\n", k, a.columns, b.columns)
		}
		mms(z, v, v2, fmt.Sprintf("value mismatch on key %s on %v and %v", k, a.columns, b.columns))
	}
}

func tpn(z *testing.T, tl *Tasklist, entryText string, queryText string, entry *Entry) {
	mme(z, tl.ParseNew(entryText, queryText), entry)
}

func SetupSearchStuff(tl *Tasklist) {
	tl.Add(tl.ParseNew("#id=10#bla questa è una prova #blo", ""))
	tl.Add(tl.ParseNew("#id=11#bib=10 ging bong un #bla", ""))
	tl.Add(tl.ParseNew("#id=12#bib=20#bla questa è una prova", ""))

	tl.Add(tl.ParseNew("#id=13#2010-01-01 bang", ""))
	tl.Add(tl.ParseNew("#id=14#2010-10-10 bang", ""))

	tl.Add(tl.ParseNew("#id=15#bza bung", ""))
	tl.Add(tl.ParseNew("#id=16#bzo bung", ""))
	tl.Add(tl.ParseNew("#id=17#bzi bung", ""))
}

func ooc() *Tasklist {
	tl :=  OpenOrCreate("/tmp/testing.pooch")
	tl.Truncate()
	SetupSearchStuff(tl)

	return tl
}

func TestSimpleEntry(z *testing.T) {
	fmt.Printf("TestSimpleEntry")
	tl := ooc()
	defer tl.Close()

	tpn(z, tl, "prova prova @blap", "",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": ""}))

	tpn(z, tl, "prova prova @blip = blop anta", "",
		MakeEntry("", "prova prova anta", "", NOW, nil, "",
		map[string]string{"uncat": "", "blip": "blop"}))

}

func TestColEntry(z *testing.T) {
	fmt.Printf("TestColEntry")
	tl := ooc()
	defer tl.Close()

	tpn(z, tl, "prova prova #+\nblip: blop\nblap:\n", "",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": "", "blip": "blop"}))

	tpn(z, tl, "prova prova #+\nblip: blop\n", "",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"uncat": "", "blip": "blop"}))
}

func TestSpecialEntry(z *testing.T) {
	fmt.Printf("TestSpecialEntry")
	tl := ooc()
	defer tl.Close()

	tpn(z, tl, "prova prova #id=ciao", "",
		MakeEntry("ciao", "prova prova", "", NOW, nil, "",
		map[string]string{"uncat": ""}))

	tpn(z, tl, "#l prova prova", "",
		MakeEntry("", "prova prova", "", LATER, nil, "",
		map[string]string{"uncat": ""}))

	tpn(z, tl, "#blap#l prova prova", "",
		MakeEntry("", "prova prova", "", LATER, nil, "",
		map[string]string{"blap": ""}))

	t, _ := ParseDateTime("2010-10-01", 0)
	tpn(z, tl, "#2010-10-01 #blap prova prova", "",
		MakeEntry("", "prova prova", "", TIMED, t, "",
		map[string]string{"blap": ""}))
}

func TestEntryWithSearch(z *testing.T) {
	fmt.Printf("TestEntryWithSearch")
	tl := ooc()
	defer tl.Close()

	tpn(z, tl, "prova prova", "prova #blap",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": ""}))

	tpn(z, tl, "prova prova", "prova #blap#blop",
		MakeEntry("", "prova prova", "", NOW, nil, "",
		map[string]string{"blap": "", "blop": ""}))
}

func tis(z *testing.T, tl *Tasklist, input string, expectedOutput string) {
	output, _, _, _, _, _, _, err := tl.ParseSearch(input, nil)
	Must(err)
	mms_large(z, output, SELECT_HEADER + expectedOutput + "\nGROUP BY tasks.id\nORDER BY priority, trigger_at_field ASC, sort DESC", "")
	stmt, err := tl.conn.Prepare("EXPLAIN " + output)
	Must(err)
	defer stmt.Finalize()
	Must(stmt.Exec())
}

func TestNoQuerySelect(z *testing.T) {
	fmt.Printf("TestNoQuerySelect")
	tl := ooc()
	defer tl.Close()

	tis(z, tl, "", "\nWHERE\n   priority <> 5")

	tis(z, tl, "#bib", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   priority <> 5")

	tis(z, tl, "#l", "\nWHERE\n   priority = 2")
	tis(z, tl, "#2010-10-2", "\nWHERE\n   trigger_at_field = '2010-10-02 00:00'\nAND\n   priority <> 5")

	tis(z, tl, "#bib#l", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   priority = 2")

	tis(z, tl, "#bib#bab#bob", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   id IN (SELECT id FROM columns WHERE name = 'bab')\nAND\n   id IN (SELECT id FROM columns WHERE name = 'bob')\nAND\n   priority <> 5")

	tis(z, tl, "#bib#bab#2010-10-02", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   id IN (SELECT id FROM columns WHERE name = 'bab')\nAND\n   trigger_at_field = '2010-10-02 00:00'\nAND\n   priority <> 5")
}

func TestExclusionSelect(z *testing.T) {
	fmt.Printf("TestExclusionSelect")
	tl := ooc()
	defer tl.Close()

	tis(z, tl, "-#bib", "\nWHERE\n   priority <> 5\nAND\n   id NOT IN (SELECT id FROM columns WHERE name = 'bib')")

	tis(z, tl, "#bib -#bab", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   priority <> 5\nAND\n   id NOT IN (SELECT id FROM columns WHERE name = 'bab')")

	tis(z, tl, "#bib -#bab -#bob", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')\nAND\n   priority <> 5\nAND\n   id NOT IN (SELECT id FROM columns WHERE name = 'bab')\nAND\n   id NOT IN (SELECT id FROM columns WHERE name = 'bob')")
}

func TestOptionsSelect(z *testing.T) {
	fmt.Printf("TestOptionsSelect")
	tl := ooc()
	defer tl.Close()

	tis(z, tl, "#bib#:w/done", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bib')")
}

func TestSavedSearchSelect(z *testing.T) {
	fmt.Printf("TestSavedSearch")
	tl := ooc()
	defer tl.Close()

	tis(z, tl, "#%idontexist", "\nWHERE\n   priority <> 5")
}

func TestQuerySelect(z *testing.T) {
	fmt.Printf("TestQuerySelect")
	tl := ooc()
	defer tl.Close()

	tis(z, tl, "prova prova #bla#blo", "\nWHERE\n   id IN (SELECT id FROM columns WHERE name = 'bla')\nAND\n   id IN (SELECT id FROM columns WHERE name = 'blo')\nAND\n   priority <> 5\nAND\n   id IN (\n      SELECT id FROM ridx WHERE title_field MATCH 'prova prova'\n   UNION\n      SELECT id FROM ridx WHERE text_field MATCH 'prova prova')")
}


func tsearch(z *testing.T, tl *Tasklist, queryText string, expectedIds []string) {
	ids := make(map[string]string)

	for _, id := range expectedIds { ids[id] = "" }

	theselect, code, _, _, _, _, _, err := tl.ParseSearch(queryText, nil)
	Must(err)
	entries, err := tl.Retrieve(theselect, code)
	Must(err)

	if len(entries) != len(ids) {
		z.Errorf("Wrong number of entries in result %d (expected %d)\nSELECT:\n%s", len(entries), len(ids), theselect)
	}

	for _, entry := range entries {
		if _, ok := ids[entry.Id()]; !ok {
			z.Errorf("Unexpected id %s in result\nSELECT:\n%s", entry.Id(), theselect)
		}
	}
}

func TestSearch(z *testing.T) {
	fmt.Printf("TestSearch")
	tl := ooc()
	defer tl.Close()

	tsearch(z, tl, "#bla", []string{ "10", "11", "12" })
	tsearch(z, tl, "#bib=10", []string{ "11" })
	tsearch(z, tl, "prova", []string{ "10", "12" })
	tsearch(z, tl, "prova #blo", []string{ "10" })
}

func TestLuaSelect(z *testing.T) {
	tl := ooc()
	defer tl.Close()

	tsearch(z, tl, "prova #+ idq('10')", []string{ "10" })
	tsearch(z, tl, "borva #+ idq('10')", []string{ })
	tsearch(z, tl, "#+ idq('10')", []string{ "10" })
	tsearch(z, tl, "#+ titleq('=', 'ging bong un')", []string{ "11" })

	tsearch(z, tl, "bang", []string{ "13", "14" })
	tsearch(z, tl, "bang #+ whenq('>', 1275775200)", []string{ "14" })
	tsearch(z, tl, "bang #+ whenq('<', 1275775200)", []string{ "13" })

	tsearch(z, tl, "#+ titleq('match', 'prova')", []string{ "10", "12" })
	tsearch(z, tl, "#+ searchq('prova')", []string{ "10", "12" })

	tsearch(z, tl, "#+ columnq('bla')", []string{ "10", "11", "12" })
	tsearch(z, tl, "#+ columnq('bib', '=', '10')", []string{ "11" })
	tsearch(z, tl, "prova #+ columnq('blo')", []string{ "10" })

	tsearch(z, tl, "bung", []string{ "15", "16", "17" })
	tsearch(z, tl, "bung #+ orq(columnq('bza'), columnq('bzo'))", []string{ "15", "16" })
	tsearch(z, tl, "bung #+ orq(columnq('bza'), idq('17'))", []string{ "15", "17" })

	tsearch(z, tl, "bung #+ notq(orq(columnq('bza'), columnq('bzo')))", []string{ "17" })

	theselect, _, _, _, _, _, _, err := tl.ParseSearch("prova #+ orq(columnq('blap', '>', 'burp'), whenq('>', 1275775200))", nil)
	Must(err)
	fmt.Printf("%s \n", theselect)
}

