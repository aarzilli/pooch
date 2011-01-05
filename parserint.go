package main

import (
	"time"
	"fmt"
	"strings"
	"os"
	"io/ioutil"
)

// part of the parser that interfaces with the backend

func (tl *Tasklist) ParseEx(text string) (*ParseResult, *Parser) {
	t := NewTokenizer(text)
	p := NewParser(t, tl.GetTimezone())
	return p.ParseEx(), p
}

func SortFromTriggerAt(triggerAt *time.Time) string {
	if triggerAt != nil {
		return triggerAt.Format("2006-01-02")
	}
	
	return time.UTC().Format("2006-01-02")
}

func ExtractColumnsFromSearch(search *ParseResult) Columns {
	cols := make(Columns)

	for _, expr := range search.include.subExpr {
		sexpr := expr.(*SimpleExpr)
		if sexpr == nil { continue }
		if sexpr.name[0] == '!' { continue }
		switch sexpr.op {
		case "=":
			cols[sexpr.name] = sexpr.value
		case "":
			cols[sexpr.name] = ""
		default:
			return nil
		}
	}

	return cols
}

func (tl *Tasklist) ParseNew(entryText, queryText string) *Entry {
	parsed, p := tl.ParseEx(entryText)

	// the following is ignored, we try to always succeed
	//if p.savedSearch != "" { return nil, MakeParseError("Saved search (@%) expression not allowed in new entry") }

	var triggerAt *time.Time = nil
	priority := NOW
	cols := make(Columns)
	id := ""

	catFound := false
	
	for _, expr := range parsed.include.subExpr {
		sexpr := expr.(*SimpleExpr)
		if sexpr == nil { continue }
		switch sexpr.name {
		case ":when": triggerAt = sexpr.valueAsTime
		case ":priority": priority = sexpr.priority
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

	// extraction of columns from search expression
	searchParsed, _ := tl.ParseEx(queryText)
	searchCols := ExtractColumnsFromSearch(searchParsed)
	if searchCols != nil {
		for k, v := range searchCols {
			cols[k] = v;
			if v == "" { catFound = true }
		}
	}

	// extra field parsing
	extraCols, extraCatFound := ParseCols(p.extra, p.timezone)
	if extraCatFound { catFound = true }
	for k, v := range extraCols { cols[k] = v }

	if id == "" { id = tl.MakeRandomId() }
	if !catFound { cols["uncat"] = "" }
	sort := SortFromTriggerAt(triggerAt)

	if triggerAt != nil { priority = TIMED }

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
	
	count := len(expr.subExpr)

	nextdepth := "   " + depth
	nnextdepth := "   " + nextdepth

	hasPriorityClause := false

	// scanning for :priority and :when fields
	for _, subExpr := range expr.subExpr {
		ssubExpr := subExpr.(*SimpleExpr)
		if ssubExpr == nil { continue} 
		if ssubExpr.name[0] != ':' { continue }
		if ssubExpr.name == ":priority" { hasPriorityClause = true }
		r = append(r, ssubExpr.IntoClause(tl, nextdepth, negate))
		count--
	}

	// scanning for complex subqueries
	for _, subExpr := range expr.subExpr {
		if subExpr.(*SimpleExpr) != nil { continue }
		r = append(r, subExpr.IntoClause(tl, nextdepth, negate))
		count--
	}

	colExprs := make([]string, 0)

	// scanning for normal 
	for _, subExpr := range expr.subExpr {
		ssubExpr := subExpr.(*SimpleExpr)
		if ssubExpr.name[0] == ':' { continue }
		if count == 1 {
			r = append(r, ssubExpr.IntoClause(tl, nextdepth, negate))
		} else {
			colExprs = append(colExprs, ssubExpr.IntoSelect(tl, nnextdepth))
		}
	}

	if len(colExprs) > 0 {
		s := "id IN (\n"
		if negate { s = "id NOT IN (\n" }
		setop := ""
		if expr.operator == "AND" {
			if negate {
				setop = "UNION"
			} else {
				setop = "INTERSECT"
			}
		} else {
			setop = "UNION"
		}
		r = append(r, nextdepth + s + strings.Join(colExprs, "\n"+nextdepth+setop + "\n") + ")")
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
	
func (parser *Parser) GetLuaClause(tl *Tasklist, pr *ParseResult) (string, os.Error) {
	if parser.extra == "" { return "", nil }
	
	tl.mutex.Lock()
	defer tl.mutex.Unlock()

	tl.SetTasklistInLua()
	tl.ResetLuaFlags()

	if tl.luaState.LoadString("return " + parser.extra) != 0 {
		errorMessage := tl.luaState.ToString(-1)
		tl.LogError(fmt.Sprintf("Error while loading lua code: %s", errorMessage))
		return "", MakeParseError(fmt.Sprintf("Error while loading lua code: %s", errorMessage))
	}

	if tl.luaState.PCall(0, 1, 0) != 0 {
		errorMessage := tl.luaState.ToString(-1)
		tl.LogError(fmt.Sprintf("Error while executing lua code: %s", errorMessage))
		return "", MakeParseError(fmt.Sprintf("Error while executing lua code: %s", errorMessage))
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

func (parser *Parser) IntoSelect(tl *Tasklist, pr *ParseResult) (string, os.Error) {
	if parser.savedSearch != "" {
		parseResult, newParser := tl.ParseEx(tl.GetSavedSearch(parser.savedSearch))
		return newParser.IntoSelect(tl, parseResult)
	}
	
	_, addDone := parser.options["w/done"]; addDone = !addDone
	where := pr.include.IntoClauses(tl, "", false, addDone)
	whereNot := pr.exclude.IntoClauses(tl, "", true, false)

	if pr.text != "" {
		where = append(where, fmt.Sprintf("   id IN (\n      SELECT id FROM ridx WHERE title_field MATCH %s\n   UNION\n      SELECT id FROM ridx WHERE text_field MATCH %s)", tl.Quote(pr.text), tl.Quote(pr.text)))
	}

	for _, v := range whereNot { where = append(where, v) }

	extraClause, error := parser.GetLuaClause(tl, pr)
	if extraClause != "" {
		where = append(where, extraClause)
	}

	whereStr := ""
	
	if len(where) > 0 {
		whereStr = "\nWHERE\n" + strings.Join(where, "\nAND\n")
	}

	return SELECT_HEADER + whereStr + "\nGROUP BY tasks.id\nORDER BY priority, trigger_at_field ASC, sort DESC", error
}


func (tl *Tasklist) ParseSearch(queryText string) (string, string, bool, os.Error) {
	pr, parser := tl.ParseEx(queryText)
	theselect, err := parser.IntoSelect(tl, pr)
	return theselect, parser.command, parser.savedSearch != "", err
}

func (tl *Tasklist) ExtendedAddParse() *Entry {
	buf, err := ioutil.ReadAll(os.Stdin)
	must(err)
	input := string(buf)

	return tl.ParseNew(input, "")
}
