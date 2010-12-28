package main

import (
	"strings"
	"strconv"
	"time"
	"regexp"
	"fmt"
)

type Parser struct {
	tkzer *Tokenizer
	showCols []string
	timezone int
	options map[string]string
	savedSearch string
	extra string
}

func NewParser(tkzer *Tokenizer, timezone int) *Parser {
	p := &Parser{tkzer, make([]string, 0), timezone, make(map[string]string), "", ""}
	tkzer.parser = p
	return p
}

type SimpleExpr struct {
	name string
	op string  // if empty string this is a simple tag expression
	
	value string
	valueAsTime *time.Time
	priority Priority

	ignore bool // if this is set the caller to ParseSimpleExpression should ignore the returned result (despite the fact that the parsing was successful)
	
	extra string
	// if the name starts with a ":" this old an extra value which is:
	// - freq for ":when"
}

func (se *SimpleExpr) String() string {
	return "#<" + se.name + ">" + "<" + se.op + se.value + ">" + "ignore=" + fmt.Sprintf("%v", se.ignore);
}

type AndExpr struct {
	subExpr []*SimpleExpr
}

type ParseResult struct {
	text string
	include AndExpr
	exclude AndExpr
}

func (p *Parser) ParseSpeculative(fn func()bool) bool {
	pos := p.tkzer.next
	r := false
	defer func() {
		if !r { p.tkzer.next = pos }
	}()
	
	r = fn()
	return r
}

func (p *Parser) ParseToken(token string) bool {
	return p.ParseSpeculative(func()bool {
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
	"<": true,
	">": true,
	"=": true,
	"<=": true,
	">=": true,
	"!=": true,
}

func (p *Parser) ParseOperationSubexpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func()bool {
		op := p.tkzer.Next()
		Logf(TRACE, "Parsed operator: [%s]\n", op)
		if _, ok := OPERATORS[op]; ok {
			Logf(TRACE, "I am looking for value\n")
			p.ParseToken(" ")
			value := p.tkzer.Next()
			if value == "" { return false }
			(*r).op = op
			(*r).value = value
			return true
		} 
		return false
	})
}

func ParseFreqToken(text string) bool {
	switch text {
	case "daily": return true
	case "weekly": return true
	case "biweekly": return true
	case "monthly": return true
	case "yearly": return true
	}
	_, err := strconv.Atoi(text)
	if err == nil { return true }
	return false
}

func ParsePriority(prstr string) Priority {
	priority := INVALID
	
	switch prstr {
	case "later", "l": priority = LATER
	case "n", "now": priority = NOW
	case "d", "done": priority = DONE
	case "$", "N", "Notes", "notes": priority = NOTES
	case "$$", "StickyNotes", "sticky": priority = STICKY
	}

	return priority
}

func (p *Parser) AttemptPriorityExpressionTransformation(r *SimpleExpr) bool {
	priority := ParsePriority(r.name)

	if priority == INVALID { return false }
	
	r.name = ":priority"
	r.priority = priority
	r.value = "see priority"
	r.op = "="
	
	return true
}

func (p *Parser) AttemptTimeExpressionTransformation(r *SimpleExpr) bool {
	split := strings.Split(r.name, "+", 2)

	parsed, err := ParseDateTime(split[0], p.timezone)
	if err != nil { return false }

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
}

func (p *Parser) AttemptSpecialTagTransformations(r *SimpleExpr) bool {
	if p.AttemptPriorityExpressionTransformation(r) { return true; }
	if p.AttemptTimeExpressionTransformation(r) { return true; }
	return false;
}

func (p *Parser) ParseOption(r *SimpleExpr) bool {
	return p.ParseSpeculative(func()bool {
		if p.tkzer.Next() != "#:" { return false }
		r.name = p.tkzer.Next()
		return true
	})
}

func (p *Parser) ParseSavedSearch(r *SimpleExpr) bool {
	return p.ParseSpeculative(func()bool {
		if p.tkzer.Next() != "#%" { return false }
		r.name = p.tkzer.Next()
		return true
	})
}

func (p *Parser) ParseSimpleExpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func()bool {
		if p.tkzer.Next() != "#" { return false }

		tagName := p.tkzer.Next()
		if !isTagChar(([]int(tagName))[0]) { return false }

		isShowCols := false
		if p.ParseToken("!") {
			isShowCols = true
		}

		if p.ParseToken("?") {
			isShowCols = true
			r.ignore = true
		}

		p.ParseToken(" ") // semi-optional space token
		hasOpSubexpr := p.ParseOperationSubexpression(r)

		r.name = tagName
		
		isSpecialTag := false
		if !hasOpSubexpr && !isShowCols {
			isSpecialTag = p.AttemptSpecialTagTransformations(r)
		}

		if !isSpecialTag {
			if isShowCols {
				p.showCols = append(p.showCols, tagName)
			}
		}

		return true
	})
}

func (p *Parser) ParseExclusion(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if !p.ParseToken("-") { return false }
		if !p.ParseSimpleExpression(r) { return false }
		return true
	})
}

func (p *Parser) ParseEx() *ParseResult {
	r := &ParseResult{}
	query := make([]string, 0)
	r.include.subExpr = make([]*SimpleExpr, 0)
	r.exclude.subExpr = make([]*SimpleExpr, 0)

LOOP: for {
		simple := &SimpleExpr{}
		switch {
		case p.ParseToken(""):
			break LOOP
		case p.ParseToken(" "):
			//nothing
		case p.ParseSavedSearch(simple):
			p.savedSearch = simple.name
		case p.ParseOption(simple):
			p.options[simple.name] = ""
		case p.ParseExclusion(simple):
			if !simple.ignore {
				r.exclude.subExpr = append(r.exclude.subExpr, simple)
			}
		case p.ParseSimpleExpression(simple):
			if !simple.ignore {
				r.include.subExpr = append(r.include.subExpr, simple)
			}
		default:
			next := p.tkzer.Next()
			if next == "@@" { next = "@" }
			if next == "##" { next = "#" }
			query = append(query, next)
		}
	}

	r.text = strings.Join([]string(query), " ")
	
	return r
}

var startMultilineRE *regexp.Regexp = regexp.MustCompile("^[ \t\n\r]*{$")
var numberRE *regexp.Regexp = regexp.MustCompile("^[0-9.]+$")

func isNumber(tk string) (n float, ok bool) {
	if !numberRE.MatchString(tk) { return -1, false }
	n, err := strconv.Atof(tk)
	if err != nil { return -1, false }
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
	for _, v := range strings.Split(colStr, "\n", -1) {
		if multilineKey != "" {
			if v == "}" {
				cols[multilineKey] = multilineValue
				Logf(DEBUG, "Adding [%s] -> multiline\n", multilineKey)
				multilineKey, multilineValue  = "", ""
			} else {
				multilineValue += v + "\n"
			}
		} else {
			vs := strings.Split(v, ":", 2)
			
			if len(vs) == 0 { continue }
			
			if len(vs) == 1 {
				// it's a category
				x := strings.TrimSpace(v)
				Logf(DEBUG, "Adding [%s]\n", x)
				if x != "" {
					cols[x] = ""
					foundcat = true
				}
			} else {
				// it (may) be a column
				key := strings.TrimSpace(vs[0])
				value := strings.TrimSpace(vs[1])

				if key == "" { continue }
				
				if startMultilineRE.MatchString(value) {
					multilineKey = key
				} else {
					value = normalizeValue(value, timezone)
					Logf(DEBUG, "Adding [%s] -> [%s]\n", key, value)
					cols[key] = value
					if value == "" { foundcat = true }
				}
			}
		}
	}

	return cols, foundcat
}
