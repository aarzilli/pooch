package main

import (
	"fmt"
	"os"
)

func FileList(pathToDir string) []os.FileInfo {
	dir, operr := os.Open(pathToDir, os.O_RDONLY, 0666)

	if operr != nil {
		panic(fmt.Sprintf("Error opening directory %s: %s", pathToDir, operr))
	}
	
	defer dir.Close()

	infos, rderr := dir.Readdir(-1)

	if rderr != nil {
		panic(fmt.Sprintf("Error reding directory %s: %s", pathToDir, rderr))
	}

	return infos
}
