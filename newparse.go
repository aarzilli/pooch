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
	valueIsNumber bool
	value string
	numValue float // only present when the value is a number
	op string  // if empty string this is a simple tag expression
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

func (p *Parser) ParseOperationSubexpression(r **SimpleExpr) bool {
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

func (p *Parser) ParseSimpleExpression(r **SimpleExpr) bool {
	return p.ParseSpeculative(func()bool {
		if p.tkzer.Next() != "#" { return false }
		tagName := p.tkzer.Next()
		if !isTagChar(([]int(tagName))[0]) { return false }

		expr := &SimpleExpr{tagName, false, "", 0, ""}

		isShowCols := false
		if p.ParseToken("!") {
			isShowCols = true
		}

		hadSpace := p.ParseToken(" ") // semi-optional space token
		wasLastToken := p.ParseToken("") 

		if !p.ParseOperationSubexpression(&expr) {
			Logf(TRACE, "Parse of operation subexpression failed\n")
			// either there was a subexpression or this must end with 
			if !hadSpace && !wasLastToken { return false }
		}

		if isShowCols {
			p.showCols.Push(tagName)
		}

		*r = expr;

		return true
	})
}