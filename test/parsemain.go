/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package parsetest

import (
	"fmt"
	"io/ioutil"
	"os"
	"main"
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
