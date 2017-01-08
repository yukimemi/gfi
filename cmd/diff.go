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
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

type info struct {
	path  string
	index int
	full  string
	diff  FileInfoValue
	value string
}

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Diff output json file",
	Long: `Diff output json file created by gfi command
and usage of using command. For example:

	gfi diff path/to/one.json path/to/other.json

`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		ci, err := GetCmdInfo()
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

		wg := new(sync.WaitGroup)
		q := make(chan info)
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
							full:  "Count",
							diff:  Count,
							value: fmt.Sprint(one.Count),
						}
					}
				}

				// Diff fileinfo.
				for _, oneFileInfo := range one.FileInfos {
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
								}
							}
						} else {
							q <- info{
								path:  args[i],
								index: i,
								full:  oneFileInfo.Full,
								diff:  Full,
								value: oneFileInfo.Full,
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
		csvMap := make(map[string][]string)
		for info := range q {
			if _, ok := csvMap[info.full]; ok {
				csvMap[info.full][info.index+2] = info.value
			} else {
				s := make([]string, len(args)+2)
				s[0] = info.full
				s[1] = fmt.Sprint(info.diff)
				csvMap[info.full] = s
			}
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
		writer := csv.NewWriter(file)

		// Write header.
		writer.Write(append([]string{"Path","type"},args...))

		for _, v := range csvMap {
			writer.Write(v)
		}
		writer.Flush()

	},
}

func init() {
	RootCmd.AddCommand(diffCmd)
}

func findFileInfo(fis FileInfos, target FileInfo) (FileInfo, error) {

	for _, fi := range fis {
		if fi.Full == target.Full {
			return fi, nil
		}
	}
	return FileInfo{}, fmt.Errorf("Not found. [%s]", target.Full)
}
