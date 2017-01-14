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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/spf13/cobra"
	"github.com/yukimemi/core"
)

var (
	sorts string
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
	Short: "Diff output json file",
	Long: `Diff output json file created by gfi command
and usage of using command. For example:

	gfi diff path/to/one.json path/to/other.json

`,
	Run: executeDiff,
}

func init() {
	RootCmd.AddCommand(diffCmd)

	// Sort with fullpath for josn.
	diffCmd.Flags().StringVarP(&sorts, "sorts", "s", "0,1", "Sort target column number with commma sepalated (ex: 1,2,0)")
}

func executeDiff(cmd *cobra.Command, args []string) {

	var (
		err    error
		ci     Cmd
		wg     *sync.WaitGroup
		q      chan info
		csvMap map[string][]string
		match  *regexp.Regexp
		ignore *regexp.Regexp
	)

	if len(args) == 0 {
		cmd.Help()
		return
	}

	var a []string
	for _, v := range args {
		files, err := filepath.Glob(v)
		if err != nil {
			log.Println(err)
			return
		}
		a = append(a, files...)
	}
	args = a

	// Recheck args.
	if len(args) <= 1 {
		cmd.Help()
		return
	}

	ci, err = GetCmdInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occur when get cmd information. [%s]\n", err)
		return
	}

	// Load json file and store.
	var loads []Output
	for _, jsonPath := range args {
		buf, err := ioutil.ReadFile(jsonPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occur when read json file. [%s]\n", err)
			return
		}
		var load Output
		err = json.Unmarshal(buf, &load)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occur when Unmarshal json file. [%s][%s]\n", jsonPath, err)
			return
		}
		loads = append(loads, load)
	}

	// Compile if given matches and ignores.
	if len(matches) != 0 {
		match, err = core.CompileStrs(matches)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compile error (%v) [%s]\n", matches, err)
			return
		}
	}
	if len(ignores) != 0 {
		ignore, err = core.CompileStrs(ignores)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Compile error (%v) [%s]\n", ignores, err)
			return
		}
	}

	wg = new(sync.WaitGroup)
	q = make(chan info)
	for i, one := range loads {
		wg.Add(1)
		go func(i int, one Output) {
			defer wg.Done()

			// Diff count
			for j, other := range loads {
				if i == j {
					continue
				}
				if one.Count != other.Count {
					q <- info{
						path:  args[i],
						index: i,
						full:  COUNT,
						diff:  Count,
						value: fmt.Sprint(one.Count),
						ford:  "",
					}
				}
			}

			// Diff fileinfo.
			for _, oneFileInfo := range one.FileInfos {
				if fileOnly && oneFileInfo.Type == DIR {
					continue
				}
				if dirOnly && oneFileInfo.Type == FILE {
					continue
				}

				// Ignore check.
				if ignore != nil && ignore.MatchString(oneFileInfo.Full) {
					continue
				}

				// Match check.
				if match != nil && !match.MatchString(oneFileInfo.Full) {
					continue
				}

				for j, other := range loads {
					if i == j {
						continue
					}

					// Get other's same full path info.
					otherFileInfo, err := findFileInfo(other.FileInfos, oneFileInfo)
					if err == nil {
						// Diff Time.
						if oneFileInfo.Time != otherFileInfo.Time {
							q <- info{
								path:  args[i],
								index: i,
								full:  oneFileInfo.Full,
								diff:  Time,
								value: oneFileInfo.Time.Format("2006/01/02 15:04:05.000"),
								ford:  oneFileInfo.Type,
							}
						}
						// Diff Size.
						if oneFileInfo.Size != otherFileInfo.Size {
							q <- info{
								path:  args[i],
								index: i,
								full:  oneFileInfo.Full,
								diff:  Size,
								value: oneFileInfo.Size,
								ford:  oneFileInfo.Type,
							}
						}
						// Diff Mode.
						if oneFileInfo.Mode != otherFileInfo.Mode {
							q <- info{
								path:  args[i],
								index: i,
								full:  oneFileInfo.Full,
								diff:  Mode,
								value: oneFileInfo.Mode,
								ford:  oneFileInfo.Type,
							}
						}
					} else {
						q <- info{
							path:  args[i],
							index: i,
							full:  oneFileInfo.Full,
							diff:  Full,
							value: oneFileInfo.Full,
							ford:  oneFileInfo.Type,
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
	csvMap = make(map[string][]string)
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
	now := time.Now()
	csvPath := func() string {
		if out != "" {
			return out
		}
		return filepath.Join(ci.Cwd, now.Format("20060102-150405.000")+".csv")
	}()

	file, err := os.Create(csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occur at create [%s] csv file. [%s]\n", csvPath, err)
		return
	}
	defer file.Close()
	writer := csv.NewWriter(transform.NewWriter(file, japanese.ShiftJIS.NewEncoder()))

	// Write header.
	writer.Write(append([]string{"key", "ford", "type"}, args...))

	// map to array.
	var csvArray records
	for _, v := range csvMap {
		csvArray = append(csvArray, v)
	}

	// sort
	sort.Sort(csvArray)

	for _, v := range csvArray {
		writer.Write(v)
	}
	writer.Flush()
	fmt.Printf("Write to [%s] file.\n", csvPath)
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
