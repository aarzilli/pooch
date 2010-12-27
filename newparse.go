package main

import (
	"unicode"
	"strings"
	"strconv"
//	"fmt"
)

type TokenizerFunc func(t *Tokenizer) (string, int);

type Tokenizer struct {
	input []int
	i int
	rewindBuffer []string
	next int
	toktable []TokenizerFunc
	parser *Parser
}

var standardTokTable []TokenizerFunc = []TokenizerFunc{
	RepeatedTokenizerTo(unicode.IsSpace, " "),
	ExtraSeparatorTokenizer,
	StrTokenizer("+"),
	StrTokenizer("-"),
	StrTokenizer("#"),
	StrTokenizerTo("@", "#"),
	StrTokenizer("%"),
	StrTokenizer("?"),
	StrTokenizer("=~"),
	StrTokenizer(">="),
	StrTokenizer("<="),
	StrTokenizer("!~"),
	StrTokenizer("!="),
	StrTokenizer("!"),
	StrTokenizer("<"),
	StrTokenizer(">"),
	RepeatedTokenizer(isTagChar),
	RepeatedTokenizer(anyChar),
}	

func NewTokenizer(input string) *Tokenizer {
	return &Tokenizer{ []int(input), 0, make([]string, 0), 0, standardTokTable, nil }
}

func anyChar(ch int) bool {
	return !unicode.IsSpace(ch)
}

func isTagChar(ch int) bool {
	if unicode.IsLetter(ch) { return true; }	
	if unicode.IsDigit(ch) { return true; }
	if ch == '-' { return true; }
	if ch == '/' { return true; }
	if ch == ',' { return true; }
	if ch == '_' { return true; }
	if ch == ':' { return true; }
	return false;
}

func StrTokenizerTo(match string, translation string) TokenizerFunc {
	umatch := []int(match)
	return func(t *Tokenizer) (string, int) {
		var j int
		for j = 0; (j < len(match)) && (t.i+j < len(t.input)) && (t.input[t.i+j] == umatch[j]); j++ { }
		if j >= len(match) {
			return translation, j
		}
		return "", 0
	}
}

func ExtraSeparatorTokenizer(t *Tokenizer) (string, int) {
	if t.i+1 >= len(t.input) { return "", 0 }
	if (t.input[t.i] != '#') && (t.input[t.i] != '@') { return "", 0 }
	if t.input[t.i+1] != '!' { return "", 0 }

	extra := string(t.input[t.i+2:])

	t.PushExtra(extra)

	return "", len(extra)+2
}

func StrTokenizer(match string) TokenizerFunc {
	return StrTokenizerTo(match, match)
}

func RepeatedTokenizer(fn func(int)bool) TokenizerFunc {
	return func(t *Tokenizer) (string, int) {
		var j int
		for j = 0; (t.i+j < len(t.input)) && fn(t.input[t.i+j]); j++ { }
		return string(t.input[t.i:t.i+j]), j
	}
}

func RepeatedTokenizerTo(fn func(int)bool, translation string) TokenizerFunc {	
	return func(t *Tokenizer) (string, int) {
		var j int
		for j = 0; (t.i+j < len(t.input)) && fn(t.input[t.i+j]); j++ { }
		return translation, j
	}
}

func (t *Tokenizer) Rest() string {
	return string(t.input[t.i:len(t.input)])
}

func (t *Tokenizer) Next() string {
	if t.next < len(t.rewindBuffer) {
		r := t.rewindBuffer[t.next]
		t.next++
		return r
	}

	r := t.RealNext()
	t.rewindBuffer = append(t.rewindBuffer, r)
	t.next++
	return r
}

func (t *Tokenizer) RealNext() string {
	if t.i >= len(t.input) { return "" }

	for _, fn := range t.toktable {
		if r, skip := fn(t); skip > 0 {
			//fmt.Printf("Matched [%s] Skipping: %d\n", r, skip);
			t.i += skip
			return r
		}
	}

	panic("Can not tokenize string")

	return ""
}

func (t *Tokenizer) PushExtra(extra string) {
	t.parser.extra = extra
}

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

	priority Priority

	ignore bool // if this is set the caller to ParseSimpleExpression should ignore the returned result (despite the fact that the parsing was successful)
	
	extra string
	// if the name starts with a "!" this old an extra value which is:
	// - freq for "!when"
}

func (se *SimpleExpr) String() string {
	return "#<" + se.name + ">" + "<" + se.op + se.value + ">";
}

type AndExpr struct {
	subExpr []*SimpleExpr
}

type BoolExpr struct {
	ored []*AndExpr
	removed []*SimpleExpr
	query string
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
	"!~": true,
	"=~": true,
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

func (p *Parser) ParseFreqToken(pfreq *string) bool {
	return p.ParseSpeculative(func()bool {
		Logf(TRACE, "Attempting to parse frequency (after a timetag)\n")
		*pfreq = p.tkzer.Next()
		switch *pfreq {
		case "daily": return true
		case "weekly": return true
		case "biweekly": return true
		case "monthly": return true
		case "yearly": return true
		}
		_, err := strconv.Atoi(*pfreq)
		if err == nil { return true }
		return false
	})
}

func (p *Parser) AttemptOptionTransformation(r *SimpleExpr) bool {
	if r.name[0] != ':' { return false }

	p.options[r.name[1:]] = r.name[1:]
	r.ignore = true

	return true
}

func (p *Parser) AttemptPriorityExpressionTransformation(r *SimpleExpr) bool {
	priority := INVALID

	switch r.name {
	case "later", "l": priority = LATER
	case "n", "now": priority = NOW
	case "d", "done": priority = DONE
	case "$", "N", "Notes", "notes": priority = NOTES
	case "$$", "StickyNotes", "sticky": priority = STICKY
	}

	if priority == INVALID { return false }
	
	r.name = "!priority"
	r.priority = priority
	r.value = "see priority"
	r.op = "="
	
	return true
}

func (p *Parser) AttemptTimeExpressionTransformation(r *SimpleExpr) bool {
	parsed, err := ParseDateTime(r.name, p.timezone)
	if err != nil { return false }
	r.name = "!when"
	r.value = parsed.Format(TRIGGER_AT_FORMAT)
	r.op = "="

	if p.ParseToken("+") {
		freq := ""
		if p.ParseFreqToken(&freq) {
			r.extra = freq
		}
	}
	
	return true
}

func (p *Parser) AttemptSpecialTagTransformations(r *SimpleExpr) bool {
	if p.AttemptOptionTransformation(r) { return true; }
	if p.AttemptPriorityExpressionTransformation(r) { return true; }
	if p.AttemptTimeExpressionTransformation(r) { return true; }
	return false;
}

func (p *Parser) ParseSimpleExpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func()bool {
		if p.tkzer.Next() != "#" { return false }

		savedSearch := false
		if p.ParseToken("%") {
			savedSearch = true
		}
			
		tagName := p.tkzer.Next()
		if !isTagChar(([]int(tagName))[0]) { return false }

		if savedSearch {
			p.savedSearch = tagName
			r.ignore = true
			return true
		}

		isShowCols := false
		if p.ParseToken("!") {
			isShowCols = true
		}

		if p.ParseToken("?") {
			isShowCols = true
			r.ignore = true
		}

		hadSpace := p.ParseToken(" ") // semi-optional space token
		wasLastToken := p.ParseToken("")
		startOfASimpleExpression := p.LookaheadToken("#") || p.LookaheadToken("+")

		hasOpSubexpr := p.ParseOperationSubexpression(r)
		
		if !hasOpSubexpr {
			Logf(TRACE, "Parse of operation subexpression failed\n")
			// either there was a subexpression or this must end with 
			if !hadSpace && !wasLastToken && !startOfASimpleExpression { return false }
		}

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

func (p *Parser) ParseAndExpr(r *AndExpr) bool {
	return p.ParseSpeculative(func() bool {
		r.subExpr = make([]*SimpleExpr, 0)

		for {
			expr := &SimpleExpr{}
			if !p.ParseSimpleExpression(expr) { break }
			p.ParseToken(" ") // optional separation space (it is not parsed by ParseSimpleExpression when there is a value involved
			if !expr.ignore {
				r.subExpr = append(r.subExpr, expr)
			}
		}

		if len(r.subExpr) == 0 { return false }

		return true
	})
}

func (p *Parser) ParseBoolExclusion(r *SimpleExpr) bool {
	return p.ParseSpeculative(func() bool {
		if !p.ParseToken("-") { return false }
		if !p.ParseSimpleExpression(r) { return false }
		return true
	})
}

func (p *Parser) ParseBoolOr(r *AndExpr) bool {
	return p.ParseSpeculative(func() bool {
		p.ParseToken("+")
		if !p.ParseAndExpr(r) { return false }
		return true
	})
}

func (p *Parser) ParseExtraSeparator() bool {
	return p.ParseSpeculative(func() bool {
		if !p.ParseToken("#") { return false }
		if !p.ParseToken("!") { return false }
		return true
	})
}

func (p *Parser) ParseNew() *AndExpr {
	r := &AndExpr{}
	query := make([]string, 0)

LOOP: for {
		switch {
		case p.ParseToken(""):
			break LOOP
		case p.ParseAndExpr(r):
			//nothing
		case p.ParseToken(" "):
			//nothing
		default:
			query = append(query, p.tkzer.Next())			
		}
	}

	return r
}

func (p *Parser) Parse() *BoolExpr {
	r := &BoolExpr{}

	r.ored = make([]*AndExpr, 0)
	r.removed = make([]*SimpleExpr, 0)
	query := make([]string, 0)

LOOP: for {
		simpleSubExpr := &SimpleExpr{}
		andSubExpr := &AndExpr{}
		
		switch {
		case p.ParseToken(""):
			break LOOP
		case p.ParseBoolExclusion(simpleSubExpr):
			if !simpleSubExpr.ignore {
				r.removed = append(r.removed, simpleSubExpr)
			}
		case p.ParseBoolOr(andSubExpr):
			r.ored = append(r.ored, andSubExpr)
		case p.ParseToken(" "):
			//nothing
		default:
			query = append(query, p.tkzer.Next())
		}
	}

	r.query = strings.Join([]string(query), " ")

	return r
}

