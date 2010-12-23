package main

import (
	"unicode"
	"strings"
//	"fmt"
)

type TokenizerFunc func(t *Tokenizer) (string, int);

type Tokenizer struct {
	input []int
	i int
	rewindBuffer []string
	next int
	toktable []TokenizerFunc
}

var standardTokTable []TokenizerFunc = []TokenizerFunc{
	RepeatedTokenizerTo(unicode.IsSpace, " "),
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
	return &Tokenizer{ []int(input), 0, make([]string, 0), 0, standardTokTable }
}

func anyChar(ch int) bool {
	return !unicode.IsSpace(ch)
}

func isTagChar(ch int) bool {
	if unicode.IsLetter(ch) { return true; }	
	if unicode.IsDigit(ch) { return true; }
	if ch == '-' { return true; }
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

type Parser struct {
	tkzer *Tokenizer
	showCols []string
}

func NewParser(tkzer *Tokenizer) *Parser {
	return &Parser{tkzer, make([]string, 0)}
}

type SimpleExpr struct {
	name string
	op string  // if empty string this is a simple tag expression
	value string
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

func (p *Parser) ParseSimpleExpression(r *SimpleExpr) bool {
	return p.ParseSpeculative(func()bool {
		if p.tkzer.Next() != "#" { return false }
		tagName := p.tkzer.Next()
		if !isTagChar(([]int(tagName))[0]) { return false }

		isShowCols := false
		if p.ParseToken("!") {
			isShowCols = true
		}

		hadSpace := p.ParseToken(" ") // semi-optional space token
		wasLastToken := p.ParseToken("")
		startOfASimpleExpression := p.LookaheadToken("#")

		if !p.ParseOperationSubexpression(r) {
			Logf(TRACE, "Parse of operation subexpression failed\n")
			// either there was a subexpression or this must end with 
			if !hadSpace && !wasLastToken && !startOfASimpleExpression { return false }
		}

		r.name = tagName
		if isShowCols {
			p.showCols = append(p.showCols, tagName)
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
			r.subExpr = append(r.subExpr, expr)
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

func (p *Parser) Parse() *BoolExpr {
	r := &BoolExpr{}

	r.ored = make([]*AndExpr, 0)
	r.removed = make([]*SimpleExpr, 0)
	query := make([]string, 0)

	for {
		simpleSubExpr := &SimpleExpr{}
		if p.ParseBoolExclusion(simpleSubExpr) {
			r.removed = append(r.removed, simpleSubExpr)
			continue
		}

		andSubExpr := &AndExpr{}
		if p.ParseBoolOr(andSubExpr) {
			r.ored = append(r.ored, andSubExpr)
			continue
		}

		if p.ParseToken("") { break }
		if !p.ParseToken(" ") { query = append(query, p.tkzer.Next()) } // either reads a space as the next token or saves it
	}

	r.query = strings.Join([]string(query), " ")

	return r
}