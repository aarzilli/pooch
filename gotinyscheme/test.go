package main

import (
	"fmt"
	"gotinyscheme"
)

func main() {
	s := gotinyscheme.NewScheme()
	defer s.Close()
	fmt.Printf("ciao %v\n", s.Eval("(tracing 1)(gocall)").String())
}