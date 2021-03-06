/*
 This program is distributed under the terms of GPLv3
 Copyright 2010-2012, Alessandro Arzilli
*/

package pooch

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SimpleExpr struct {
	name string
	op   string // if empty string this is a simple tag expression

	value       string
	valueAsTime *time.Time
	priority    Priority

	extra string
	// if the name starts with a ":" this old an extra value which is:
	// - freq for ":when"
}

func (se *SimpleExpr) String() string {
	return "#<" + se.name + ">" + "<" + se.op + se.value + ">"
}

type Clausable interface {
	IntoClause(tl *Tasklist, depth string, negate bool) string
}

type BoolExpr struct {
	operator string
	subExpr  []Clausable
}

type ParseResult struct {
	text    string
	include BoolExpr
	exclude BoolExpr
	options map[string]string

	savedSearch string
	extra       string // text after the #+ separator
	command     string // text after the #! separator

	showCols []string

	timezone int
}

func MakeParseResult() *ParseResult {
	r := &ParseResult{}

	r.include.operator = "AND"
	r.exclude.operator = "AND"
	r.include.subExpr = make([]Clausable, 0)
	r.exclude.subExpr = make([]Clausable, 0)

	r.options = make(map[string]string)

	return r
}

type Parser struct {
	tkzer    *Tokenizer
	timezone int
	result   *ParseResult
}

func NewParser(tkzer *Tokenizer, timezone int) *Parser {
	p := &Parser{tkzer, timezone, MakeParseResult()}
	p.result.timezone = timezone
	tkzer.parser = p
	return p
}

func (p *Parser) ParseSpeculative(fn func() bool) bool {
	pos := p.tkzer.next
	r := false
	defer func() {
		if !r {
			p.tkzer.next = pos
		}
	}()

	r = fn()
	return r
}

func (p *Parser) ParseToken(token string) bool {
	return p.ParseSpeculative(func() bool {
		return p.tkzer.Next() == token
	})
}

func (p *Parser) LookaheadToken(token string) bool {
	pos := p.tkzer.next
	r := p.tkzer.Next() == token
	p.tkzer.next = pos
	return r
}

var OPERATORS map[string]bool = map[string]bool{
	"<":  true,
	">":  true,
	"=":  true,
	"<=": true,
	">=": true,
	"!=": true,
}

func (p *Parser) ParseOperationSubexpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		op := p.tkzer.Next()
		Logf(TRACE, "Parsed operator: [%s]\n", op)
		if _, ok := OPERATORS[op]; ok {
			Logf(TRACE, "I am looking for value\n")
			p.ParseToken(" ")
			value := p.tkzer.Next()
			if value == "" {
				return false
			}
			(*r).op = op
			(*r).value = value
			return true
		}
		return false
	})
}

func ParseFreqToken(text string) bool {
	switch text {
	case "daily":
		return true
	case "weekly":
		return true
	case "biweekly":
		return true
	case "monthly":
		return true
	case "yearly":
		return true
	}
	_, err := strconv.Atoi(text)
	if err == nil {
		return true
	}
	return false
}

func ParsePriority(prstr string) Priority {
	priority := INVALID

	switch prstr {
	case "later", "l":
		priority = LATER
	case "n", "now":
		priority = NOW
	case "d", "done":
		priority = DONE
	case "$", "N", "Notes", "notes":
		priority = NOTES
	case "$$", "StickyNotes", "sticky":
		priority = STICKY
	case "timed":
		priority = TIMED
	}

	return priority
}

func (p *Parser) ParseOption(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if p.tkzer.Next() != "#:" {
			return false
		}
		r.name = p.tkzer.Next()
		if p.ParseToken("=") {
			r.op = "="
			r.value = p.tkzer.Next()
		} else {
			r.value = ""
		}
		return true
	})
}

func (p *Parser) ParseSavedSearch(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if p.tkzer.Next() != "#%" {
			return false
		}
		r.name = p.tkzer.Next()
		return true
	})
}

func (p *Parser) ParseColumnRequest() bool {
	return p.ParseSpeculative(func() bool {
		if p.tkzer.Next() != "#" {
			return false
		}

		colName := p.tkzer.Next()
		if !isTagChar(([]rune(colName))[0]) {
			return false
		}

		if p.tkzer.Next() != "?" {
			return false
		}
		p.result.showCols = append(p.result.showCols, colName)

		return true
	})
}

func (p *Parser) ParsePriorityExpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if p.tkzer.Next() != "#" {
			return false
		}
		tag := p.tkzer.Next()

		priority := ParsePriority(tag)

		if priority == INVALID {
			return false
		}

		r.name = ":priority"
		r.priority = priority
		r.value = "see priority"
		r.op = "="

		return true
	})
}

func (p *Parser) ParseTimeExpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if p.tkzer.Next() != "#" {
			return false
		}

		timeExpr := p.tkzer.Next()

		split := strings.SplitN(timeExpr, "+", 2)

		parsed, err := ParseDateTime(split[0], p.timezone)
		if err != nil {
			return false
		}

		freq := ""
		if len(split) > 1 {
			freq = split[1]
			if !ParseFreqToken(freq) {
				return false
			}
		}

		r.name = ":when"
		r.valueAsTime = parsed
		r.value = parsed.Format(TRIGGER_AT_FORMAT)
		r.op = "="
		r.extra = freq

		return true
	})
}

func (p *Parser) ParseSimpleExpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if p.tkzer.Next() != "#" {
			return false
		}

		tagName := p.tkzer.Next()
		if !isTagChar(([]rune(tagName))[0]) {
			return false
		}

		isShowCols := false
		if p.ParseToken("!") {
			isShowCols = true
		}

		p.ParseToken(" ") // semi-optional space token
		p.ParseOperationSubexpression(r)

		r.name = tagName

		if isShowCols {
			p.result.showCols = append(p.result.showCols, tagName)
		}

		return true
	})
}

func (p *Parser) ParseExclusion(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if !p.ParseToken("-") {
			return false
		}
		if !p.ParseSimpleExpression(r) {
			return false
		}
		return true
	})
}

func (p *Parser) ParseEx() *ParseResult {
	query := make([]string, 0)

LOOP:
	for {
		simple := &SimpleExpr{}
		switch {
		case p.ParseToken(""):
			break LOOP
		case p.ParseToken(" "):
			if len(query)-1 >= 0 {
				if query[len(query)-1] != " " {
					query = append(query, " ")
				}
			}
		case p.ParseSavedSearch(simple):
			p.result.savedSearch = simple.name
		case p.ParseOption(simple):
			if simple.value == "" {
				p.result.options[simple.name] = ""
			} else {
				simple.name = ":" + simple.name
				p.result.include.subExpr = append(p.result.include.subExpr, simple)
			}
		case p.ParseColumnRequest():
			// nothing to do
		case p.ParseExclusion(simple):
			p.result.exclude.subExpr = append(p.result.exclude.subExpr, simple)
		case p.ParsePriorityExpression(simple):
			p.result.include.subExpr = append(p.result.include.subExpr, simple)
		case p.ParseTimeExpression(simple):
			p.result.include.subExpr = append(p.result.include.subExpr, simple)
		case p.ParseSimpleExpression(simple):
			p.result.include.subExpr = append(p.result.include.subExpr, simple)
		default:
			next := p.tkzer.Next()
			if next == "@@" {
				next = "@"
			}
			if next == "##" {
				next = "#"
			}
			query = append(query, next)
		}
	}

	p.result.text = strings.TrimSpace(strings.Join([]string(query), ""))

	return p.result
}

var startMultilineRE *regexp.Regexp = regexp.MustCompile("^[ \t\n\r]*{$")
var numberRE *regexp.Regexp = regexp.MustCompile("^[0-9.]+$")

func isNumber(tk string) (n float64, ok bool) {
	if !numberRE.MatchString(tk) {
		return -1, false
	}
	n, err := strconv.ParseFloat(tk, 64)
	if err != nil {
		return -1, false
	}
	return n, true
}

func normalizeValue(value string, timezone int) string {
	Logf(DEBUG, "Normalizing: [%s]\n", value)
	if t, _ := ParseDateTime(value, timezone); t != nil {
		value = t.Format(TRIGGER_AT_FORMAT)
	} else if n, ok := isNumber(value); ok {
		value = fmt.Sprintf("%0.6f", n)
	}

	return value
}

func ParseCols(colStr string, timezone int) (Columns, bool) {
	cols := make(Columns)

	multilineKey := ""
	multilineValue := ""

	foundcat := false
	for _, v := range strings.Split(colStr, "\n") {
		if multilineKey != "" {
			if v == "}" {
				cols[multilineKey] = multilineValue
				Logf(DEBUG, "Adding [%s] -> multiline\n", multilineKey)
				multilineKey, multilineValue = "", ""
			} else {
				multilineValue += v + "\n"
			}
		} else {
			if len(v) < 2 {
				continue
			}
			isSpecial := false
			if v[0] == ':' {
				isSpecial = true
				v = v[1:]
			}
			vs := strings.SplitN(v, ":", 2)

			if len(vs) == 0 {
				continue
			}

			if len(vs) == 1 {
				// it's a category
				x := strings.TrimSpace(v)
				Logf(DEBUG, "Adding [%s]\n", x)
				if x != "" {
					if isSpecial {
						x = ":" + x
					}
					cols[x] = ""
					foundcat = true
				}
			} else {
				// it (may) be a column
				key := strings.TrimSpace(vs[0])
				value := strings.TrimSpace(vs[1])

				if key == "" {
					continue
				}

				if strings.HasPrefix(key, "sub/") {
					foundcat = true
				}

				if startMultilineRE.MatchString(value) {
					multilineKey = key
				} else {
					if isSpecial {
						key = ":" + key
					}
					// I don't really need this at the moment
					//value = normalizeValue(value, timezone)
					Logf(DEBUG, "Adding [%s] -> [%s]\n", key, value)
					cols[key] = value
					if value == "" {
						foundcat = true
					}
				}
			}
		}
	}

	return cols, foundcat
}

func ParseTsvFormat(in string, tl *Tasklist, timezone int) *Entry {
	fields := strings.SplitN(in, "\t", 4)

	entry := tl.ParseNew(fields[1], "")

	priority := ParsePriority(fields[2])

	var triggerAt *time.Time = nil
	var sort string
	if priority == TIMED {
		triggerAt, _ = ParseDateTime(fields[3], timezone)
		sort = SortFromTriggerAt(triggerAt, tl.GetSetting("defaultsorttime") == "1")
	} else {
		sort = fields[3]
	}

	entry.SetId(fields[0])
	entry.SetPriority(priority)
	entry.SetTriggerAt(triggerAt)
	entry.SetSort(sort)

	return entry
}
