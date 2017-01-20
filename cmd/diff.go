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
	"regexp"
	"sort"
	"strings"
	"sync"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/spf13/cobra"
	"github.com/yukimemi/core"
)

const (
	// DiffHeader is diff command output csv header.
	DiffHeader = "Path\tType\tDiff"
)

var (
	sjisIn bool
)

type info struct {
	path  string
	index int
	full  string
	diff  FileInfoValue
	value string
	ford  string
}

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff output csv file",
	Long: `Diff output csv file created by gfi command
and usage of using command. For example:

	gfi diff path/to/one.csv path/to/other.csv

`,
	Run: executeDiff,
}

func init() {
	RootCmd.AddCommand(diffCmd)

	// Sort with target column for csv.
	diffCmd.Flags().StringVarP(&sorts, "sorts", "s", "0,2", "Sort target column number with commma sepalated (ex: 1,2,0)")
	// Whether input csv in ShiftJIS encoding.
	diffCmd.Flags().BoolVarP(&sjisIn, "sjisin", "J", false, "Input csv in ShiftJIS encoding")
}

func executeDiff(cmd *cobra.Command, args []string) {

	var (
		err error

		match  *regexp.Regexp
		ignore *regexp.Regexp

		csvMap  = make(map[string][]string)
		fisList = make([]FileInfos, 0)
		q       = make(chan info)
		wg      = new(sync.WaitGroup)
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

	// Recheck args.
	if len(args) <= 1 {
		cmd.Help()
		return
	}

	// Load csv and store.
	for _, csvPath := range args {
		fmt.Println("Open:", csvPath)
		c, err := os.Open(csvPath)
		if err != nil {
			log.Fatalln(err)
		}
		defer c.Close()
		var reader *csv.Reader
		if sjisIn {
			reader = csv.NewReader(transform.NewReader(c, japanese.ShiftJIS.NewDecoder()))
		} else {
			reader = csv.NewReader(c)
		}
		reader.Comma = '\t'
		// Skip header.
		_, err = reader.Read()
		if err != nil {
			log.Fatalln(err)
		}
		left, err := reader.ReadAll()
		if err != nil {
			log.Fatalln(err)
		}

		// Change data to FileInfos struct.
		fis := make(FileInfos, 0)
		for _, r := range left {
			fis = append(fis, *csvToFileInfo(r))
		}
		fisList = append(fisList, fis)
	}

	// Compile if given matches and ignores.
	if len(matches) != 0 {
		match, err = core.CompileStrs(matches)
		if err != nil {
			log.Fatalln(err)
		}
	}
	if len(ignores) != 0 {
		ignore, err = core.CompileStrs(ignores)
		if err != nil {
			log.Fatalln(err)
		}
	}

	for i, one := range fisList {
		wg.Add(1)
		go func(i int, one FileInfos) {
			defer wg.Done()

			// Diff fileinfo.
			for _, oneFi := range one {
				if fileOnly && oneFi.Type == DIR {
					continue
				}
				if dirOnly && oneFi.Type == FILE {
					continue
				}

				// Ignore check.
				if ignore != nil && ignore.MatchString(oneFi.Full) {
					continue
				}

				// Match check.
				if match != nil && !match.MatchString(oneFi.Full) {
					continue
				}

				for j, other := range fisList {
					if i == j {
						continue
					}

					// Get other's same full path info.
					otherFi, err := findFileInfo(other, oneFi)
					if err == nil {
						// Diff Time.
						if oneFi.Time != otherFi.Time {
							q <- info{
								path:  args[i],
								index: i,
								full:  oneFi.Full,
								diff:  FileTime,
								value: oneFi.Time,
								ford:  oneFi.Type,
							}
						}
						// Diff Size.
						if oneFi.Size != otherFi.Size {
							q <- info{
								path:  args[i],
								index: i,
								full:  oneFi.Full,
								diff:  FileSize,
								value: oneFi.Size,
								ford:  oneFi.Type,
							}
						}
						// Diff Mode.
						if oneFi.Mode != otherFi.Mode {
							q <- info{
								path:  args[i],
								index: i,
								full:  oneFi.Full,
								diff:  FileMode,
								value: oneFi.Mode,
								ford:  oneFi.Type,
							}
						}
					} else {
						q <- info{
							path:  args[i],
							index: i,
							full:  oneFi.Full,
							diff:  FileFull,
							value: oneFi.Full,
							ford:  oneFi.Type,
						}
					}
				}
			}
		}(i, one)
	}

	// Async wait.
	go func() {
		wg.Wait()
		close(q)
	}()

	// Receive diff and store to array.
	for info := range q {
		cnt++
		if !silent {
			fmt.Fprintf(os.Stderr, "Count: %d\r", cnt)
		}
		key := info.full + fmt.Sprint(info.diff)
		if _, ok := csvMap[key]; ok {
			csvMap[key][info.index+3] = info.value
		} else {
			s := make([]string, len(args)+3)
			s[0] = info.full
			s[1] = info.ford
			s[2] = fmt.Sprint(info.diff)
			s[info.index+3] = info.value
			csvMap[key] = s
		}
	}

	if len(csvMap) == 0 {
		fmt.Println("There is no difference !")
		return
	}

	// Output to csv.
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
	err = writer.Write(append(strings.Split(DiffHeader, "\t"), args...))
	if err != nil {
		log.Fatalln(err)
	}

	// map to array.
	var csvArray records
	for _, v := range csvMap {
		csvArray = append(csvArray, v)
	}

	// sort
	if sorts == "" {
		sorts = "0,2"
	}
	sort.Sort(csvArray)

	for _, v := range csvArray {
		err = writer.Write(v)
		if err != nil {
			log.Fatalln(err)
		}
	}
	writer.Flush()
	fmt.Printf("Write to [%s]. ([%d] row)\n", out, cnt)
}

func findFileInfo(fis FileInfos, target FileInfo) (FileInfo, error) {

	for _, fi := range fis {
		if fi.Full == target.Full {
			return fi, nil
		}
	}
	return FileInfo{}, fmt.Errorf("Not found. [%s]", target.Full)
}

func csvToFileInfo(data []string) *FileInfo {
	return &FileInfo{
		Full: data[FileFull-1],
		Rel:  data[FileRel-1],
		Abs:  data[FileAbs-1],
		Name: data[FileName-1],
		Time: data[FileTime-1],
		Size: data[FileSize-1],
		Mode: data[FileMode-1],
		Type: data[FileType-1],
	}
}
