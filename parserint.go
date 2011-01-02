package main

import (
	"time"
	"fmt"
	"strings"
	"os"
)

// part of the parser that interfaces with the backend

func ParseEx(tl *Tasklist, text string) (*ParseResult, *Parser) {
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
		if expr.name[0] == '!' { continue }
		switch expr.op {
		case "=":
			cols[expr.name] = expr.value
		case "":
			cols[expr.name] = ""
		default:
			return nil
		}
	}

	return cols
}

func (tl *Tasklist) ParseNew(entryText, queryText string) *Entry {
	parsed, p := ParseEx(tl, entryText)

	// the following is ignored, we try to always succeed
	//if p.savedSearch != "" { return nil, MakeParseError("Saved search (@%) expression not allowed in new entry") }

	var triggerAt *time.Time = nil
	priority := NOW
	cols := make(Columns)
	id := ""

	catFound := false
	
	for _, expr := range parsed.include.subExpr {
		switch expr.name {
		case ":when": triggerAt = expr.valueAsTime
		case ":priority": priority = expr.priority
		case "id": id = expr.value
		default:
			if expr.op == "" {
				cols[expr.name] = ""
				catFound = true
			} else if expr.op == "=" {
				cols[expr.name] = expr.value
			}
		}
	}

	// extraction of columns from search expression
	searchParsed, _ := ParseEx(tl, queryText)
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
		
	case ":priority":
		return fmt.Sprintf("priority = %d", expr.priority)
		
	case ":when":
		if sqlop, ok := OPERATOR_CHECK[expr.op]; ok {
			return fmt.Sprintf("trigger_at_field %s '%s'", sqlop, expr.valueAsTime.Format(TRIGGER_AT_FORMAT))
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

type Clausable interface{
	IntoClause(tl *Tasklist, depth string, negate bool) string
}

func (expr *SimpleExpr) IntoClause(tl *Tasklist, depth string, negate bool) string {
	if expr.name[0] == ':' {
		return fmt.Sprintf("%s%s", depth, expr.IntoClauseEx(tl))
	}

	s := "IN"
	if negate { s = "NOT IN" }
	return fmt.Sprintf("%sid %s (%s)", depth, s, expr.IntoClauseEx(tl))
}

func (expr *AndExpr) IntoClauses(tl *Tasklist, parser *Parser, depth string, negate bool) []string {
	r := make([]string, 0)
	
	count := len(expr.subExpr)

	nextdepth := "   " + depth
	nnextdepth := "   " + nextdepth

	hasPriorityClause := false

	// scanning for :priority and :when fields
	for _, subExpr := range expr.subExpr {
		if subExpr.name[0] != ':' { continue }
		if subExpr.name == ":priority" { hasPriorityClause = true }
		r = append(r, subExpr.IntoClause(tl, nextdepth, negate))
		count--
	}

	colExprs := make([]string, 0)

	// scanning for normal 
	for _, subExpr := range expr.subExpr {
		if subExpr.name[0] == ':' { continue }
		if count == 1 {
			r = append(r, subExpr.IntoClause(tl, nextdepth, negate))
		} else {
			colExprs = append(colExprs, subExpr.IntoSelect(tl, nnextdepth))
		}
	}

	if len(colExprs) > 0 {
		s := "id IN (\n"
		if negate { s = "id NOT IN (\n" }
		r = append(r, nextdepth + s + strings.Join(colExprs, "\n"+nextdepth+"INTERSECT\n") + ")")
	}

	if !hasPriorityClause && !negate {
		if _, ok := parser.options["w/done"]; !ok {
			r = append(r, nextdepth + "priority <> 5")
		}
	}

	return r
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

	if tl.luaState.GetTop() < 1 { fmt.Printf("no output\n"); return "", MakeParseError("Extra lua code didn't return anything") }
	switch {
	case tl.luaState.IsString(-1):
		//TODO compile string
		fmt.Printf("Got a string!: %s\n", tl.luaState.ToString(-1))
	case tl.luaState.IsLightUserdata(-1):
		ud := tl.ToGoInterface(-1)
		tl.luaState.Pop(1)
		if clausable := ud.(Clausable); clausable != nil {
			return clausable.IntoClause(tl, "   ", false), nil
		}
	}

	tl.luaState.Pop(1)

	return "", MakeParseError("Unable to interpret values returned by the extra lua code")
}

func (parser *Parser) IntoSelect(tl *Tasklist, pr *ParseResult) (string, os.Error) {
	if parser.savedSearch != "" {
		parseResult, newParser := ParseEx(tl, tl.GetSavedSearch(parser.savedSearch))
		return newParser.IntoSelect(tl, parseResult)
	}
	
	where := pr.include.IntoClauses(tl, parser, "", false)
	whereNot := pr.exclude.IntoClauses(tl, parser, "", true)

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

	return "SELECT tasks.id, title_field, text_field, priority, trigger_at_field, sort, group_concat(columns.name||':'||columns.value, '\v')\nFROM tasks NATURAL JOIN columns" + whereStr + "\nGROUP BY tasks.id\nORDER BY priority, trigger_at_field ASC, sort DESC", error
}


func (tl *Tasklist) ParseSearch(queryText string) (string, string, os.Error) {
	pr, parser := ParseEx(tl, queryText)
	theselect, err := parser.IntoSelect(tl, pr)
	return theselect, parser.command, err
}

