/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package pooch

import (
	"time"
	"fmt"
	"strings"
	"os"
	"io/ioutil"
	"sort"
)

// part of the parser that interfaces with the backend

func (tl *Tasklist) ParseEx(text string) *ParseResult {
	t := NewTokenizer(text)
	p := NewParser(t, tl.GetTimezone())
	return p.ParseEx()
}

func SortFromTriggerAt(triggerAt *time.Time, defaultWithTime bool) string {
	if triggerAt != nil {
		return triggerAt.Format("2006-01-02")
	}

	if defaultWithTime {
		return time.Now().UTC().Format("2006-01-02_15:04:05")
	}

	return time.Now().UTC().Format("2006-01-02")
}

func ExtractColumnsFromSearch(search *ParseResult) Columns {
	cols := make(Columns)

	for _, expr := range search.include.subExpr {
		sexpr, ok := expr.(*SimpleExpr)
		if !ok { continue }
		if sexpr.name[0] == '!' { continue }
		switch sexpr.op {
		case "=":
			if sexpr.name[0] != ':' && sexpr.name[0] != '%' {
				cols[sexpr.name] = sexpr.value
			}
		case "":
			if sexpr.name[0] != ':' && sexpr.name[0] != '%' {
				cols[sexpr.name] = ""
			}
		default:
			return nil
		}
	}

	return cols
}

func (tl *Tasklist) ExpandColumnsFromOntology(cols Columns) {
	io := InvertOntology([]string{}, tl.GetOntology(), []InvOntologyEntry{})

	var addcols = make(Columns)

	for k, v := range cols {
		if v != "" {
			continue
		}

		var colioe *InvOntologyEntry = nil
		for i, ioe := range io {
			if ioe.cat == k {
				if colioe == nil {
					colioe = &io[i]
				} else {
					colioe = nil
					break
				}
			}
		}

		if colioe == nil {
			continue
		}

		for _, p := range colioe.parents {
			addcols[p] = ""
		}
	}

	for k, _ := range addcols {
		cols[k] = ""
	}
}

func (tl *Tasklist) SortFromSubitems(subitemId string) string {
	n := tl.CountCategoryItems(subitemId)
	return fmt.Sprintf("%03d", n+1)
}

func (tl *Tasklist) ParseNew(entryText, queryText string) *Entry {
	parsed := tl.ParseEx(entryText)

	// the following is ignored, we try to always succeed
	//if p.savedSearch != "" { return nil, MakeParseError("Saved search (@%) expression not allowed in new entry") }

	var triggerAt *time.Time = nil
	priority := NOW
	cols := make(Columns)
	id := ""
	prioritySet := false

	catFound := false

	for _, expr := range parsed.include.subExpr {
		sexpr, ok := expr.(*SimpleExpr)
		if !ok { continue }
		switch sexpr.name {
		case ":when": triggerAt = sexpr.valueAsTime
		case ":priority":
			priority = sexpr.priority
			prioritySet = true
		case "id": id = sexpr.value
		default:
			if sexpr.op == "" {
				cols[sexpr.name] = ""
				catFound = true
			} else if sexpr.op == "=" {
				cols[sexpr.name] = sexpr.value
			}
		}
	}

	subitem, subitemId := IsSubitem(cols)

	var sort string
	if sortv, ok := cols["sort"]; ok {
		sort = sortv
		delete(cols, "sort")
	} else if subitem {
		sort = tl.SortFromSubitems(subitemId)
	} else {
		sort = SortFromTriggerAt(triggerAt, tl.GetSetting("defaultsorttime") == "1")
	}

	if !subitem {
		// extraction of columns from search expression
		searchParsed := tl.ParseEx(queryText)
		searchCols := ExtractColumnsFromSearch(searchParsed)
		if searchCols != nil {
			for k, v := range searchCols {
				cols[k] = v;
				if v == "" { catFound = true }
			}
		}

		tl.ExpandColumnsFromOntology(cols)
	}

	// extra field parsing
	extraCols, extraCatFound := ParseCols(parsed.extra, parsed.timezone)
	if extraCatFound { catFound = true }
	for k, v := range extraCols { cols[k] = v }

	if id == "" { id = tl.MakeRandomId() }
	if !catFound { cols["uncat"] = "" }


	if !prioritySet {
		if triggerAt != nil { priority = TIMED }
	}

	return MakeEntry(id, parsed.text, "", priority, triggerAt, sort, cols)
}

type NotExpr struct {
	subExpr Clausable
}

func (ne *NotExpr) IntoClause(tl *Tasklist, depth string, negate bool) string {
	return depth+"NOT (\n"+ne.subExpr.IntoClause(tl, depth+"   ", negate)+")"
}

var OPERATOR_CHECK map[string]string = map[string]string{
	"!=": "<>",
	"=": "=",
	">": ">",
	"<": "<",
	">=": ">=",
	"<=": "<=",
	"LIKE": "LIKE",
	"like": "LIKE",
	"not like": "NOT LIKE",
	"NOT LIKE": "NOT LIKE",
	"NOTLIKE": "NOT LIKE",
	"notlike": "NOT LIKE",
}

func (expr *SimpleExpr) IntoClauseEx(tl *Tasklist) string {
	switch expr.name {
	case ":id": fallthrough
	case ":title_field": fallthrough
	case ":text_field":
		if expr.op == "match" {
			return fmt.Sprintf("id IN (SELECT id FROM ridx WHERE %s MATCH %s)", expr.name[1:], tl.Quote(expr.value))
		} else if sqlop, ok := OPERATOR_CHECK[expr.op]; ok {
			return fmt.Sprintf("%s %s %s", expr.name[1:], sqlop, tl.Quote(expr.value))
		}

	case ":search":
		return fmt.Sprintf("id IN (SELECT id FROM ridx WHERE title_field MATCH %s UNION SELECT id FROM ridx WHERE text_field MATCH %s)", tl.Quote(expr.value), tl.Quote(expr.value))

	case ":priority":
		return fmt.Sprintf("priority = %d", expr.priority)

	case ":when":
		if expr.op == "notnull" {
			return "trigger_at_field IS NOT NULL"
		} else if sqlop, ok := OPERATOR_CHECK[expr.op]; ok {
			value := expr.value
			if expr.valueAsTime != nil {
				value = expr.valueAsTime.Format(TRIGGER_AT_FORMAT)
			}
			return fmt.Sprintf("trigger_at_field %s '%s'", sqlop, value)
		} else {
			panic(MakeParseError(fmt.Sprintf("Unknown operator %s", expr.op)))
		}

	default:
		if expr.name[0] == ':' {
			panic(MakeParseError(fmt.Sprintf("Unknown pseudo-field %s", expr.name)))
		}

		if expr.op == "" {
			return fmt.Sprintf("SELECT id FROM columns WHERE name = %s", tl.Quote(expr.name))
		} else if sqlop, ok := OPERATOR_CHECK[expr.op]; ok {
			return fmt.Sprintf("SELECT id FROM columns WHERE name = %s AND value %s %s", tl.Quote(expr.name), sqlop, tl.Quote(expr.value))
		} else {
			panic(MakeParseError(fmt.Sprintf("Unknown operator %s", expr.op)))
		}
	}

	panic(MakeParseError("Something bad happened"))
}

func (expr *SimpleExpr) IntoSelect(tl *Tasklist, depth string) string {
	if expr.name[0] == ':' {
		return fmt.Sprintf("%sSELECT id FROM tasks WHERE %s", depth, expr.IntoClauseEx(tl))
	}

	return fmt.Sprintf("%s%s", depth, expr.IntoClauseEx(tl))
}

func (expr *SimpleExpr) IntoClause(tl *Tasklist, depth string, negate bool) string {
	if expr.name[0] == ':' {
		return fmt.Sprintf("%s%s", depth, expr.IntoClauseEx(tl))
	}

	s := "IN"
	if negate { s = "NOT IN" }
	return fmt.Sprintf("%sid %s (%s)", depth, s, expr.IntoClauseEx(tl))
}

func (expr *BoolExpr) IntoClauses(tl *Tasklist, depth string, negate bool, addDone bool) []string {
	r := make([]string, 0)

	nextdepth := "   " + depth

	hasPriorityClause := false

	for _, subExpr := range expr.subExpr {
		if ssubExpr, ok := subExpr.(*SimpleExpr); ok {
			if ssubExpr.name == ":priority" {
				hasPriorityClause = true
			}
		}

		r = append(r, subExpr.IntoClause(tl, nextdepth, negate))
	}

	if !hasPriorityClause && addDone {
		r = append(r, nextdepth + "priority <> 5")
	}

	return r
}

func (expr *BoolExpr) IntoClause(tl *Tasklist, depth string, negate bool) string {
	clauses := expr.IntoClauses(tl, depth+"   ", negate, false)
	return depth + "(\n" + strings.Join(clauses, "\n"+depth+expr.operator+"\n") + ")"

}

func (pr *ParseResult) GetLuaClause(tl *Tasklist) (string, error) {
	if pr.extra == "" { return "", nil }

	tl.SetTasklistInLua()
	tl.ResetLuaFlags()

	tl.luaState.CheckStack(1)
	if tl.luaState.LoadString("return " + pr.extra) != 0 {
		errorMessage := tl.luaState.ToString(-1)
		tl.LogError(fmt.Sprintf("Error while loading lua code: %s", errorMessage))
		return "", MakeParseError(fmt.Sprintf("Error while loading lua code: %s", errorMessage))
	}

	//fmt.Printf("Executing: %s\n", pr.extra)
	if err := tl.luaState.Call(0, 1); err != nil {
		tl.LogError(fmt.Sprintf("Error while executing lua code: %s", err.Error()))
		return "", MakeParseError(fmt.Sprintf("Error while executing lua code: %s", err.Error()))
	}

	tl.luaState.CheckStack(1)

	if tl.luaState.GetTop() < 1 {
		return "", MakeParseError("Extra lua code didn't return anything")
	}

	clausable := GetQueryObject(tl, -1)
	if clausable == nil {
		return "", MakeParseError("Extra lua code didn't return a query object")
	}

	return clausable.IntoClause(tl, "   ", false), nil
}

func (pr *ParseResult) AddIncludeClause(expr *SimpleExpr) {
	pr.include.subExpr = append(pr.include.subExpr, expr)
}

var SELECT_HEADER string = "SELECT tasks.id, title_field, text_field, priority, trigger_at_field, sort, group_concat(columns.name||'\u001f'||columns.value, '\u001f')\nFROM tasks NATURAL JOIN columns "

func (pr *ParseResult) ResolveSavedSearch(tl *Tasklist) *ParseResult {
	if pr.savedSearch != "" {
		parseResult := tl.ParseEx(tl.GetSavedSearch(pr.savedSearch))
		return parseResult
	}
	return pr
}

func (pr *ParseResult) IntoSelect(tl *Tasklist, luaClausable Clausable) (string, map[string]string, error) {
	if pr.savedSearch != "" {
		parseResult := tl.ParseEx(tl.GetSavedSearch(pr.savedSearch))
		r, _, err := parseResult.IntoSelect(tl, luaClausable)
		return r, parseResult.options, err
	}

	_, addDone := pr.options["w/done"]; addDone = !addDone
	where := pr.include.IntoClauses(tl, "", false, addDone)
	whereNot := pr.exclude.IntoClauses(tl, "", true, false)

	if pr.text != "" {
		where = append(where, fmt.Sprintf("   id IN (\n      SELECT id FROM ridx WHERE title_field MATCH %s\n   UNION\n      SELECT id FROM ridx WHERE text_field MATCH %s)", tl.Quote(pr.text), tl.Quote(pr.text)))
	}

	for _, v := range whereNot { where = append(where, v) }

	var err error
	if luaClausable == nil {
		var extraClause string
		extraClause, err = pr.GetLuaClause(tl)
		if extraClause != "" {
			where = append(where, extraClause)
		}
	} else {
		var extraClause string
		extraClause, err = luaClausable.IntoClause(tl, "   ", false), nil
		if extraClause != "" {
			where = append(where, extraClause)
		}
	}

	whereStr := ""

	if len(where) > 0 {
		whereStr = "\nWHERE\n" + strings.Join(where, "\nAND\n")
	}

	orderBy := "ORDER BY priority, trigger_at_field ASC, sort DESC"
	if _, found := pr.options["ssort"]; found {
		orderBy = "ORDER BY sort ASC"
	}

	return SELECT_HEADER + whereStr + "\nGROUP BY tasks.id\n" + orderBy , nil, err
}

func (pr *ParseResult) IntoTrigger() string {
	if pr.savedSearch != "" {
		return "#%" + pr.savedSearch
	}

	if pr.extra != "" { return "" }

	out := make([]string, 0)

	for _, se := range pr.include.subExpr {
		if sse, ok := se.(*SimpleExpr); ok {
			out = append(out, sse.name)
		}
	}

	sort.Strings(out)
	return "#" + strings.Join(out, "#")
}

func (pr *ParseResult) IsEmpty() bool {
	if pr.savedSearch != "" { return false }
	if pr.command != "" { return false }
	if pr.extra != "" { return false }
	if pr.text != "" { return false }
	if len(pr.exclude.subExpr) > 0 { return false }
	if len(pr.include.subExpr) > 0 { return false }
	return true
}

func (tl *Tasklist) ParseSearch(queryText string, luaClausable Clausable) (string, string, string, bool, bool, []string, map[string]string, error) {
	pr := tl.ParseEx(queryText)
	isEmpty := pr.IsEmpty()
	theselect, extraOptions, err := pr.IntoSelect(tl, luaClausable)
	trigger := pr.IntoTrigger()

	options := make(map[string]string)
	for k, v := range extraOptions {
		options[k] = v
	}
	for k, v := range pr.options {
		options[k] = v
	}

	return theselect, pr.command, trigger, pr.savedSearch != "", isEmpty, pr.showCols, options, err
}

func (tl *Tasklist) ExtendedAddParse() *Entry {
	buf, err := ioutil.ReadAll(os.Stdin)
	Must(err)
	input := string(buf)

	split1 := strings.SplitN(input, "\n", 2)
	split2 := strings.SplitN(split1[1], "\n@+\n", 2)

	toParse := split1[0]

	if len(split2) > 1 {
		toParse += "\n@+\n" + split2[1]
	}

	e := tl.ParseNew(toParse, "")
	e.SetText(split2[0])

	return e;
}
