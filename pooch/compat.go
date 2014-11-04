/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
*/

package pooch

import (
	"bufio"
	"os"
	"text/tabwriter"
)

func CompatHelp() {
	tw := tabwriter.NewWriter(os.Stderr, 8, 8, 4, ' ', 0)
	w := bufio.NewWriter(tw)

	w.WriteString("nothing here at the moment")

	w.Flush()
	tw.Flush()
}
