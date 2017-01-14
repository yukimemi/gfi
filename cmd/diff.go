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
	"strconv"
	"strings"
	"sync"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/spf13/cobra"
	"github.com/yukimemi/core"
)

var (
	sorts  string
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

type records [][]string

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

	// Sort with fullpath for josn.
	diffCmd.Flags().StringVarP(&sorts, "sorts", "s", "0,1", "Sort target column number with commma sepalated (ex: 1,2,0)")
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
	args, err = getGlobArgs(args)
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
								diff:  Time,
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
								diff:  Size,
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
								diff:  Mode,
								value: oneFi.Mode,
								ford:  oneFi.Type,
							}
						}
					} else {
						q <- info{
							path:  args[i],
							index: i,
							full:  oneFi.Full,
							diff:  Full,
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
	err = writer.Write(append([]string{"Key", "Type", "Diff"}, args...))
	if err != nil {
		log.Fatalln(err)
	}

	// map to array.
	var csvArray records
	for _, v := range csvMap {
		csvArray = append(csvArray, v)
	}

	// sort
	sort.Sort(csvArray)

	for _, v := range csvArray {
		err = writer.Write(v)
		if err != nil {
			log.Fatalln(err)
		}
	}
	writer.Flush()
	fmt.Printf("Write to [%s] file.\n", out)
}

func findFileInfo(fis FileInfos, target FileInfo) (FileInfo, error) {

	for _, fi := range fis {
		if fi.Full == target.Full {
			return fi, nil
		}
	}
	return FileInfo{}, fmt.Errorf("Not found. [%s]", target.Full)
}

// Len returns records length.
func (r records) Len() int {
	return len(r)
}

// Less returns which record is less.
func (r records) Less(i, j int) bool {
	indexes := strings.Split(sorts, ",")
	for _, index := range indexes {
		ii, err := strconv.Atoi(index)
		if err != nil {
			log.Fatalln(err)
		}
		if r[i][ii] < r[j][ii] {
			return true
		} else if r[i][ii] > r[j][ii] {
			return false
		}
	}
	return false
}

// Swap is records swap func.
func (r records) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func csvToFileInfo(data []string) *FileInfo {
	return &FileInfo{
		Full: data[Full-1],
		Rel:  data[Rel-1],
		Abs:  data[Abs-1],
		Name: data[Name-1],
		Time: data[Time-1],
		Size: data[Size-1],
		Mode: data[Mode-1],
		Type: data[Type-1],
	}
}
