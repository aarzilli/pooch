package main

import (
	"time"
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

