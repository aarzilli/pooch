package main

type Tokenizer struct {
	input string
	i int
}

func NewTokenizer(input string) *Tokenizer {
	return &Tokenizer{input, 0}
}

func (t *Tokenizer) Next() string {
	if t.input[t.i] == ' ' {
		for t.input[t.i] == ' ' {
			t.i++
		}
		return " "
	} else if t.input[t.i] == '+' {
		return "+"
	} else if t.input[t.i] == '-' {
		return "-"
	} else if (t.input[t.i] == '#') || (t.input[t.i] == '@') {
		//TODO: only accept letters, numbers and dash (how?)

		
		//TODO: read tag
	}

	return ""
}