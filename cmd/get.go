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
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/yukimemi/file"
)

// Cmd options.
var (
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
	if len(args) == 0 {
		cmd.Help()
		return
	}
	ci, err := GetCmdInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occur when get cmd information. [%s]\n", err)
		return
	}

	var fis FileInfos
	wg := new(sync.WaitGroup)
	for _, root := range args {
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			fi, err := getFileInfo(root)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error occur when get [%s] directory file information. [%s]\n", root, err)
				return
			}
			fis = append(fis, fi...)
		}(root)
	}
	wg.Wait()
	if len(fis) == 0 {
		return
	}
	// sort FileInfos if sort flag set.
	if sortFlg {
		sort.Sort(fis)
	}
	now := time.Now()
	jsonPath := func() string {
		if out != "" {
			return out
		}
		return filepath.Join(ci.Cwd, now.Format("20060102-150405.000")+".json")
	}()
	// Add Count info.
	output := Output{
		Count:     len(fis),
		FileInfos: fis,
	}
	j, err := json.MarshalIndent(output, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occur at MarshalIndent for [%v]. [%s]\n", fis, err)
		return
	}
	os.MkdirAll(filepath.Dir(jsonPath), os.ModePerm)
	file, err := os.Create(jsonPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occur at create [%s] json file. [%s]\n", jsonPath, err)
		return
	}
	defer file.Close()
	n, err := file.Write(j)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error occur at write [%s] json file. [%s]\n", jsonPath, err)
		return
	}
	fmt.Printf("Write to [%s] file. ([%d] bytes)\n", jsonPath, n)
}

func getFileInfo(root string) (FileInfos, error) {

	var (
		fis FileInfos
		fs  chan file.Info
		cnt int
		err error
	)

	opt := file.Option{
		Matches: matches,
		Ignores: ignores,
		Recurse: true,
	}

	if fileOnly && !dirOnly {
		fs, err = file.GetFiles(root, opt)
	} else if !fileOnly && dirOnly {
		fs, err = file.GetDirs(root, opt)
	} else {
		fs, err = file.GetFilesAndDirs(root, opt)
	}
	if err != nil {
		return nil, err
	}

	cnt = 0
	for f := range fs {
		cnt++
		fmt.Fprintf(os.Stderr, "Count: \r%d            \r", cnt)
		if f.Err != nil {
			if errSkip {
				fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", f.Err)
				continue
			}
			return nil, f.Err
		}
		abs, err := filepath.Abs(f.Path)
		if err != nil {
			if errSkip {
				fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", err)
				continue
			}
			return nil, err
		}
		full, err := filepath.Abs(file.ShareToAbs(f.Path))
		if err != nil {
			if errSkip {
				fmt.Fprintf(os.Stderr, "Warning: [%s]. continue.\n", err)
				continue
			}
			return nil, err
		}
		info := FileInfo{
			Full: full,
			Abs:  abs,
			Rel:  f.Path,
			Name: f.Fi.Name(),
			Time: f.Fi.ModTime(),
			Size: fmt.Sprint(f.Fi.Size()),
			Mode: f.Fi.Mode().String(),
			Type: getType(f.Fi),
		}
		fis = append(fis, info)
	}

	return fis, err
}

func getType(f os.FileInfo) string {
	if f.IsDir() {
		return DIR
	}
	return FILE
}
