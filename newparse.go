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

