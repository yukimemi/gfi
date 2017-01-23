// Copyright © 2017 yukimemi
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
	"github.com/yukimemi/core"
	"github.com/yukimemi/file"
)

// SizeInfo is directory size info.
type SizeInfo struct {
	path string
	size int
}

// sizeCmd represents the size command
var sizeCmd = &cobra.Command{
	Use:   "size path/to/dir",
	Short: "Get directory size",
	Long: `Get directory size command.
For example:

	gfi size path/to/dir

`,
	Run: executeSize,
}

func init() {
	RootCmd.AddCommand(sizeCmd)

	// Skip flag.
	sizeCmd.Flags().BoolVarP(&errSkip, "err", "e", false, "Skip getting directory information on error")
	// Sort with target column for csv.
	sizeCmd.Flags().StringVarP(&sorts, "sorts", "s", "", "Sort target column number with commma sepalated (ex: 1,2,0)")
}

func executeSize(cmd *cobra.Command, args []string) {

	var (
		err error

		di       = make(chan DirInfo)
		csvArray = make(records, 0)
		wg       = new(sync.WaitGroup)
	)

	if len(args) == 0 {
		cmd.Help()
		return
	}
	// Get glob file args.
	args, err = core.GetGlobArgs(args)
	if err != nil {
		log.Fatalln(err)
	}

	for _, root := range args {
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			err = getDirInfo(root, di)
			if err != nil {
				log.Fatalln(err)
			}
		}(root)
	}

	// Async wait.
	go func() {
		wg.Wait()
		close(di)
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
	err = writer.Write(getDirCsvHeader())
	if err != nil {
		log.Fatalln(err)
	}

	// Receive and output.
	for d := range di {
		cnt++
		if !silent {
			fmt.Fprintf(os.Stderr, "Count: %d\r", cnt)
		}
		if sorts != "" {
			csvArray = append(csvArray, dirInfoToCsv(d))
		} else {
			err = writer.Write(dirInfoToCsv(d))
			if err != nil {
				log.Fatalln(err)
			}
		}
	}

	// sort FileInfos if sort flag set.
	if sorts != "" {
		sort.Sort(csvArray)
		for _, v := range csvArray {
			err = writer.Write(v)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}
	writer.Flush()
	if cnt == 0 {
		fmt.Println("There is no information to get.")
		c.Close()
		os.RemoveAll(out)
	} else {
		fmt.Printf("Write to [%s]. ([%d] row)\n", out, cnt)
	}
}

func getDirInfo(root string, di chan DirInfo) error {

	var (
		err  error
		ferr error
		dirs chan file.Info
		wg   sync.WaitGroup

		opt = file.Option{
			Matches: matches,
			Ignores: ignores,
			Recurse: true,
		}
	)

	dirs, err = file.GetDirs(root, opt)

	if err != nil {
		return err
	}

	for d := range dirs {
		wg.Add(1)
		go func(d file.Info) {
			defer wg.Done()
			var dInfo DirInfo

			if d.Err != nil {
				if errSkip {
					fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", d.Err)
					return
				}
				if d.Path != "" {
					dInfo.Rel = d.Path
				}
				ferr = d.Err
				return
			}
			dInfo.Abs, err = filepath.Abs(d.Path)
			if err != nil {
				if errSkip {
					fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", err)
					return
				}
				ferr = err
				return
			}
			dInfo.Full, err = filepath.Abs(file.ShareToAbs(d.Path))
			if err != nil {
				if errSkip {
					fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", err)
					return
				}
				ferr = err
				return
			}
			t := file.GetDirInfo(d.Path)
			if t.Err != nil {
				if errSkip {
					fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", t.Err)
					return
				}
				ferr = err
				return
			}
			dInfo.Rel = d.Path
			dInfo.Name = d.Fi.Name()
			dInfo.Time = d.Fi.ModTime().Format("2006/01/02 15:04:05.000")
			dInfo.Size = fmt.Sprint(t.DirSize)
			dInfo.FileCount = t.FileCount
			dInfo.DirCount = t.DirCount
			di <- dInfo
		}(d)
	}
	wg.Wait()
	if ferr != nil {
		if errSkip {
			fmt.Fprintf(os.Stderr, "Warning: [%s].\n", ferr)
			return nil
		}
		return ferr
	}
	return nil
}

func dirInfoToCsv(di DirInfo) []string {
	a := make([]string, DirMax)
	a[DirFull-1] = di.Full
	a[DirRel-1] = di.Rel
	a[DirAbs-1] = di.Abs
	a[DirName-1] = di.Name
	a[DirTime-1] = di.Time
	a[DirSize-1] = di.Size
	a[DirFileCount-1] = fmt.Sprint(di.FileCount)
	a[DirDirCount-1] = fmt.Sprint(di.DirCount)
	return a
}

func getDirCsvHeader() []string {
	var div DirInfoValue
	header := make([]string, 0)
	for div = 1; div <= DirMax; div++ {
		header = append(header, div.String())
	}
	return header
}
