package main

import (
	"time"
	"fmt"
	"strings"
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

func ParseNew(tl *Tasklist, entryText, queryText string) *Entry {
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

func (expr *SimpleExpr) IntoClauseEx(tl *Tasklist) string {
	switch expr.name {
	case ":priority":
		return fmt.Sprintf("priority = %d", expr.priority)
		
	case ":when":
		switch expr.op {
		case "!=":
			expr.op = "<>"
			fallthrough
		case "=":
			fallthrough
		case ">":
			fallthrough
		case "<":
			fallthrough
		case ">=":
			fallthrough
		case "<=":
			return fmt.Sprintf("trigger_at_field %s '%s'", expr.op, expr.valueAsTime.Format(TRIGGER_AT_FORMAT))
		default:
			panic(MakeParseError(fmt.Sprintf("Unknown operator %s", expr.op)))
		}

	default:
		if expr.name[0] == ':' {
			panic(MakeParseError(fmt.Sprintf("Unknown pseudo-field %s", expr.name)))
		}
		
		switch expr.op {
		case "":
			return fmt.Sprintf("SELECT id FROM columns WHERE name = %s", tl.Quote(expr.name))
		case "!=":
			expr.op = "<>"
			fallthrough
		case "=":
			fallthrough
		case ">":
			fallthrough
		case "<":
			fallthrough
		case ">=":
			fallthrough
		case "<=":
			return fmt.Sprintf("SELECT id FROM columns WHERE name = %s AND value %s %s", tl.Quote(expr.name), expr.op, tl.Quote(expr.value))
		default:
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

func (parser *Parser) IntoSelect(tl *Tasklist, pr *ParseResult) string {
	if parser.savedSearch != "" {
		parseResult, newParser := ParseEx(tl, tl.GetSavedSearch(parser.savedSearch))
		return newParser.IntoSelect(tl, parseResult)
	}
	
	where := pr.include.IntoClauses(tl, parser, "", false)
	whereNot := pr.exclude.IntoClauses(tl, parser, "", true)

	for _, v := range whereNot { where = append(where, v) }

	whereStr := ""
	
	if len(where) > 0 {
		whereStr = "\nWHERE\n" + strings.Join(where, "\nAND\n")
	}

	return "SELECT tasks.id, title_field, text_field, priority, trigger_at_field, sort, group_concat(columns.name||':'||columns.value, '\v')\nFROM tasks NATURAL JOIN columns" + whereStr + "\nGROUP BY tasks.id\nORDER BY priority, trigger_at_field ASC, sort DESC"
	
	//TODO:
	// - do the query
}