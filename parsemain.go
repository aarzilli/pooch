package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	buf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse: error reading: %s\n", err.String())
	}
	s := string(buf)
	//fmt.Printf("Parsing: %s", s)
	e, _ := QuickParse(s)
	fmt.Printf("Parsed: %s\n", Deparse(e))
}
