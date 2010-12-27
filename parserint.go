package main

import (
	"time"
)

// part of the parser that interfaces with the backend

func SortFromTriggerAt(triggerAt *time.Time) string {
	if triggerAt != nil {
		return triggerAt.Format("2006-01-02")
	}
	
	return time.UTC().Format("2006-01-02")
}

func ParseSearchEx(tl *Tasklist, queryText string) *BoolExpr {
	t := NewTokenizer(queryText)
	p := NewParser(t, tl.GetTimezone())
	return p.Parse()
}

func ExtractColumnsFromSearch(search *BoolExpr) Columns {
	if len(search.ored) != 1 { return nil }

	andExpr := search.ored[0]
	
	cols := make(Columns)

	for _, expr := range andExpr.subExpr {
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
	t := NewTokenizer(entryText)
	p := NewParser(t, tl.GetTimezone())

	title, exprs := p.ParseNew()

	// the following is ignored, we try to always succeed
	//if p.savedSearch != "" { return nil, MakeParseError("Saved search (@%) expression not allowed in new entry") }

	var triggerAt *time.Time = nil
	priority := NOW
	cols := make(Columns)
	id := ""

	catFound := false
	
	for _, expr := range exprs.subExpr {
		switch expr.name {
		case "!when": triggerAt = expr.valueAsTime
		case "!priority": priority = expr.priority
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
	searchExpr := ParseSearchEx(tl, queryText)
	searchCols := ExtractColumnsFromSearch(searchExpr)
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

	return MakeEntry(id, title, "", priority, triggerAt, sort, cols)
}

