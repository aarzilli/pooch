package main

import (
	"unicode"
	"container/vector"
//	"fmt"
)

type TokenizerFunc func(t *Tokenizer) (string, int);

type Tokenizer struct {
	input []int
	i int
	rewindBuffer *vector.StringVector
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
	return &Tokenizer{ []int(input), 0, new(vector.StringVector), 0, standardTokTable }
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
	if t.next < t.rewindBuffer.Len() {
		r := t.rewindBuffer.At(t.next)
		t.next++
		return r
	}

	r := t.RealNext()
	t.rewindBuffer.Push(r)
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
	showCols *vector.StringVector
}

func NewParser(tkzer *Tokenizer) *Parser {
	return &Parser{tkzer, new(vector.StringVector)}
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
	ored []AndExpr
	removed []SimpleExpr
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
			p.showCols.Push(tagName)
		}

		return true
	})
}

func (p *Parser) ParseAndExpr(r *AndExpr) bool {
	return p.ParseSpeculative(func() bool {
		var subExpr vector.Vector

		for {
			expr := &SimpleExpr{}
			if !p.ParseSimpleExpression(expr) { break }
			p.ParseToken(" ") // optional separation space (it is not parsed by ParseSimpleExpression when there is a value involved
			subExpr.Push(expr)
		}

		if subExpr.Len() == 0 { return false }

		r.subExpr = make([]*SimpleExpr, subExpr.Len())
		for i, v := range subExpr { r.subExpr[i] = v.(*SimpleExpr) }
		
		return true
	})
}

func (p *Parser) Parse() *BoolExpr {
	r := &BoolExpr{}

	var ored vector.Vector
	var removed vector.Vector

	for {
		gotPlus := p.ParseToken("+")
		
		gotDash := false
		if !gotPlus {
			gotDash = p.ParseToken("-")
		}

		
	}

	//TODO:
	// - se il prossimo token e` + o - segnarselo
	// - chiamare ParseAndExpr
	// - usare il risultato insieme a + o - 
	// - se non riesce aggiungere il token alla stringa
}