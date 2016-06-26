package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var cmdDef = &Command{
	UsageLine: "gfi is get file information tool",
	Short:     "Get file info",
	Long:      "Get file information.",
	Run:       run,
}

func run(args []string) int {

	// Walk path and get file info.
	for _, root := range args {
		_ = filepath.Walk(root, func(f string, fi os.FileInfo, e error) error {
			fmt.Println(f)
			return e
		})
	}

	return 0
}
