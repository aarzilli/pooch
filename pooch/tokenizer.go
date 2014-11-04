/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
*/

package pooch

import (
	"unicode"
	//	"fmt"
)

type TokenizerFunc func(t *Tokenizer) (string, int)

type Tokenizer struct {
	input        []rune
	i            int
	rewindBuffer []string
	next         int
	toktable     []TokenizerFunc
	parser       *Parser
}

var standardTokTable []TokenizerFunc = []TokenizerFunc{
	RepeatedTokenizerTo(unicode.IsSpace, " "),
	ExtraSeparatorTokenizer,

	StrTokenizer("-"),

	// saved search tag
	StrTokenizer("#%"),
	StrTokenizerTo("@%", "#%"),

	// escaped tokens
	StrTokenizer("@@"),
	StrTokenizer("##"),

	StrTokenizer("#:"),
	StrTokenizerTo("@:", "#:"),

	// tag
	StrTokenizer("#"),
	StrTokenizerTo("@", "#"),

	// operators
	StrTokenizer("="),
	StrTokenizer(">="),
	StrTokenizer("<="),
	StrTokenizer("!="),
	StrTokenizer("<"),
	StrTokenizer(">"),

	// terminators
	StrTokenizer("!"),
	StrTokenizer("?"),

	// anything else
	RepeatedTokenizer(isTagChar),
	RepeatedTokenizer(anyChar),
}

func NewTokenizer(input string) *Tokenizer {
	return &Tokenizer{[]rune(input), 0, make([]string, 0), 0, standardTokTable, nil}
}

func anyChar(ch rune) bool {
	return !unicode.IsSpace(ch)
}

func isTagChar(ch rune) bool {
	if unicode.IsLetter(ch) {
		return true
	}
	if unicode.IsDigit(ch) {
		return true
	}
	if ch == '+' {
		return true
	}
	if ch == '-' {
		return true
	}
	if ch == '/' {
		return true
	}
	if ch == ',' {
		return true
	}
	if ch == '_' {
		return true
	}
	if ch == ':' {
		return true
	}
	return false
}

func StrTokenizerTo(match string, translation string) TokenizerFunc {
	umatch := []rune(match)
	return func(t *Tokenizer) (string, int) {
		var j int
		for j = 0; (j < len(match)) && (t.i+j < len(t.input)) && (t.input[t.i+j] == umatch[j]); j++ {
		}
		if j >= len(match) {
			return translation, j
		}
		return "", 0
	}
}

func isQuickTagStart(ch rune) bool {
	return ch == '#' || ch == '@'
}

func ConsumeRealExtra(t *Tokenizer) string {
	var j int
	for j = t.i + 2; j < len(t.input); j++ {
		if !isQuickTagStart(t.input[j]) {
			continue
		}
		if j+1 >= len(t.input) {
			continue
		}
		if t.input[j+1] != '!' {
			continue
		}

		// found @! or #! decrement j to spit out the quick tag start and exit the loop
		break
	}

	return string(t.input[t.i+2 : j])
}

func ExtraSeparatorTokenizer(t *Tokenizer) (string, int) {
	if t.i+1 >= len(t.input) {
		return "", 0
	}
	if !isQuickTagStart(t.input[t.i]) {
		return "", 0
	}

	switch t.input[t.i+1] {
	case '+':
		extra := ConsumeRealExtra(t)
		t.PushExtra(extra)
		return " ", len(extra) + 2
	case '!':
		command := string(t.input[t.i+2:])
		t.PushCommand(command)
		return "", len(command) + 2
	}

	return "", 0
}

func StrTokenizer(match string) TokenizerFunc {
	return StrTokenizerTo(match, match)
}

func RepeatedTokenizer(fn func(rune) bool) TokenizerFunc {
	return func(t *Tokenizer) (string, int) {
		var j int
		for j = 0; (t.i+j < len(t.input)) && fn(t.input[t.i+j]); j++ {
		}
		return string(t.input[t.i : t.i+j]), j
	}
}

func RepeatedTokenizerTo(fn func(rune) bool, translation string) TokenizerFunc {
	return func(t *Tokenizer) (string, int) {
		var j int
		for j = 0; (t.i+j < len(t.input)) && fn(t.input[t.i+j]); j++ {
		}
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
	if t.i >= len(t.input) {
		return ""
	}

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
	t.parser.result.extra = extra
}

func (t *Tokenizer) PushCommand(command string) {
	t.parser.result.command = command
}
