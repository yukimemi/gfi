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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

type info struct {
	path  string
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

		// TODO
		_ = ci

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
							diff:  Count,
							value: one.Count,
						}
					}

				}
				// Diff fileinfo
				for j, other := range loads {
					if i == j {
						continue
					}

					for _, oneFileInfo := range one.FileInfos {
						// Get other's same full path info.
						otherFileInfo, err := findFileInfo(other.FileInfos, oneFileInfo)
						if err == nil {
							// Diff Time.
							if oneFileInfo.Time != otherFileInfo.Time {
								q <- info{
									path:  args[i],
									diff:  Time,
									value: oneFileInfo.Time.Format("2006/01/02 15:04:05.000"),
								}
							}
							// Diff Size.
							if oneFileInfo.Size != otherFileInfo.Size {
								q <- info{
									path:  args[i],
									diff:  Size,
									value: oneFileInfo.Size,
								}
							}
							// Diff Mode.
							if oneFileInfo.Mode != otherFileInfo.Mode {
								q <- info{
									path:  args[i],
									diff:  Mode,
									value: oneFileInfo.Mode,
								}
							}
						} else {
							q <- info{path: args[i]}
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

		// Receive diff.
		for fi := range q {
			fmt.Println(fi)
		}

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
