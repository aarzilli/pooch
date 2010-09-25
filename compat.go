/*
 This program is distributed under the terms of GPLv3
 Copyright 2010, Alessandro Arzilli
 */

package main

import (
	"fmt"
	"os"
	"tabwriter"
	"bufio"
)

func CmdPort(args []string) {
	CheckArgs(args, 2, 2, "port")
	
	filename, _ := Resolve(args[0])
	
	Log(DEBUG, "Resolved filename: ", filename)

	Port(filename, args[1])
}

func HelpPort() {
	fmt.Fprintf(os.Stderr, "porting older dbs\n")
}
func CompatHelp() {
	tw := tabwriter.NewWriter(os.Stderr, 8, 8, 4, ' ', 0)
	w := bufio.NewWriter(tw)

	w.WriteString("\tport\tAdds a columns table to a database that doesn't have it and populates it with base informations\n")

	w.Flush()
	tw.Flush()
}
