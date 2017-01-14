// Copyright © 2017 yukimemi <yukimemi@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/spf13/cobra"
	"github.com/yukimemi/file"
)

var (
	// Cmd options.
	sortFlg bool
	errSkip bool
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get path/to/dir",
	Short: "Get file information",
	Long: `Get file information command. filepath, size, mode etc.
For example:

	gfi path/to/dir

`,
	Run: executeGet,
}

func init() {
	RootCmd.AddCommand(getCmd)

	// Sort with fullpath for josn.
	getCmd.Flags().BoolVarP(&sortFlg, "sort", "s", false, "Sort flag")
	// Skip flag.
	getCmd.Flags().BoolVarP(&errSkip, "err", "e", false, "Skip getting file information on error")
}

func executeGet(cmd *cobra.Command, args []string) {

	var (
		err error

		fi  = make(chan FileInfo)
		fis = make(FileInfos, 0)
		wg  = new(sync.WaitGroup)
	)

	if len(args) == 0 {
		cmd.Help()
		return
	}
	// Get glob file args.
	args, err = getGlobArgs(args)
	if err != nil {
		log.Fatalln(err)
	}

	for _, root := range args {
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			err = getFileInfo(root, fi)
			if err != nil {
				log.Fatalln(err)
			}
		}(root)
	}

	// Async wait.
	go func() {
		wg.Wait()
		close(fi)
	}()

	// Create output csv file.
	os.MkdirAll(filepath.Dir(out), os.ModePerm)
	c, err := os.Create(out)
	if err != nil {
		log.Fatalln(err)
	}
	defer c.Close()
	var writer *csv.Writer
	if sjisOut {
		writer = csv.NewWriter(transform.NewWriter(c, japanese.ShiftJIS.NewEncoder()))
	} else {
		writer = csv.NewWriter(c)
	}
	writer.Comma = '\t'
	writer.UseCRLF = true

	// Write header.
	err = writer.Write(getCsvHeader())
	if err != nil {
		log.Fatalln(err)
	}

	// Receive and output.
	for f := range fi {
		cnt++
		fmt.Fprintf(os.Stderr, "Count: %d\r", cnt)
		if sortFlg {
			fis = append(fis, f)
		} else {
			err = writer.Write(fileInfoToCsv(f))
			if err != nil {
				log.Fatalln(err)
			}
		}
	}

	// sort FileInfos if sort flag set.
	if sortFlg {
		sort.Sort(fis)
		for _, f := range fis {
			err = writer.Write(fileInfoToCsv(f))
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	writer.Flush()
	fmt.Printf("Write to [%s]. ([%d] row)\n", out, cnt)
}

func getFileInfo(root string, fi chan FileInfo) error {

	var (
		err error
		fs  chan file.Info

		opt = file.Option{
			Matches: matches,
			Ignores: ignores,
			Recurse: true,
		}
	)

	if fileOnly && !dirOnly {
		fs, err = file.GetFiles(root, opt)
	} else if !fileOnly && dirOnly {
		fs, err = file.GetDirs(root, opt)
	} else {
		fs, err = file.GetFilesAndDirs(root, opt)
	}

	if err != nil {
		return err
	}

	for f := range fs {
		if f.Err != nil {
			if errSkip {
				fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", f.Err)
				continue
			}
			return f.Err
		}
		abs, err := filepath.Abs(f.Path)
		if err != nil {
			if errSkip {
				fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", err)
				continue
			}
			return err
		}
		full, err := filepath.Abs(file.ShareToAbs(f.Path))
		if err != nil {
			if errSkip {
				fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", err)
				continue
			}
			return err
		}
		fi <- FileInfo{
			Full: full,
			Abs:  abs,
			Rel:  f.Path,
			Name: f.Fi.Name(),
			Time: f.Fi.ModTime().Format("2006/01/02 15:04:05.000"),
			Size: fmt.Sprint(f.Fi.Size()),
			Mode: f.Fi.Mode().String(),
			Type: getType(f.Fi),
		}
	}

	return err
}

func getType(f os.FileInfo) string {
	if f.IsDir() {
		return DIR
	}
	return FILE
}

func fileInfoToCsv(fi FileInfo) []string {
	a := make([]string, Max)
	a[Full-1] = fi.Full
	a[Rel-1] = fi.Rel
	a[Abs-1] = fi.Abs
	a[Name-1] = fi.Name
	a[Time-1] = fi.Time
	a[Size-1] = fi.Size
	a[Mode-1] = fi.Mode
	a[Type-1] = fi.Type
	return a
}

func getCsvHeader() []string {
	var fiv FileInfoValue
	header := make([]string, 0)
	for fiv = 1; fiv <= Max; fiv++ {
		header = append(header, fiv.String())
	}
	return header
}
