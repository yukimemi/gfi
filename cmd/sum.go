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
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"sync"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"

	"github.com/spf13/cobra"
	"github.com/yukimemi/core"
)

var (
	// Cmd options.
	del            string
	keyCol, valCol int
)

// sumCmd represents the sum command
var sumCmd = &cobra.Command{
	Use:   "sum path/to/dir",
	Short: "Output csv put together",
	Long: `Output csv put together.
For example:

	gfi sum -k 0 -v 2 path/to/one.csv path/to/two.csv

`,
	Run: executeSum,
}

type line struct {
	index int
	key   string
	value string
}

func init() {
	RootCmd.AddCommand(sumCmd)

	// Key column number..
	sumCmd.Flags().IntVarP(&keyCol, "key", "k", 0, "Key column number (default is 0)")
	// Value column number.
	sumCmd.Flags().IntVarP(&valCol, "val", "v", 1, "Value column number (default is 1)")
	// Csv delimiter.
	sumCmd.Flags().StringVarP(&del, "delimiter", "D", "\t", "Csv delimiter. (default is TAB)")
	// Sort with target column for csv.
	sumCmd.Flags().StringVarP(&sorts, "sorts", "s", "0", "Sort target column number with commma sepalated (ex: 0,1,2)")
	// Whether input csv in ShiftJIS encoding.
	sumCmd.Flags().BoolVarP(&sjisIn, "sjisin", "J", false, "Input csv in ShiftJIS encoding")
}

func executeSum(cmd *cobra.Command, args []string) {

	var (
		err error

		match   *regexp.Regexp
		ignore  *regexp.Regexp
		keyName string

		csvMap  = make(map[string][]string)
		readers = make([]*csv.Reader, 0)
		q       = make(chan line)
		wg      = new(sync.WaitGroup)
		sem     = make(chan struct{}, runtime.NumCPU())
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
		reader.Comma = []rune(del)[0]
		// Get key name.
		header, err := reader.Read()
		keyName = header[keyCol]
		if err != nil {
			log.Fatalln(err)
		}
		readers = append(readers, reader)
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

	for i, reader := range readers {
		wg.Add(1)
		go func(i int, r *csv.Reader) {
			sem <- struct{}{}
			defer func() {
				wg.Done()
				<-sem
			}()

			// Loop records.
			for {

				record, err := r.Read()
				if err == io.EOF {
					break
				}

				l := line{
					index: i,
					key:   record[keyCol],
					value: record[valCol],
				}

				// Check Ignore.
				if ignore != nil && ignore.MatchString(l.key) {
					continue
				}

				// Match check.
				if match != nil && !match.MatchString(l.key) {
					continue
				}
				q <- l
			}
		}(i, reader)
	}

	// Async wait.
	go func() {
		wg.Wait()
		close(q)
	}()

	// Receive diff and store to array.
	for line := range q {
		cnt++
		if !silent {
			fmt.Fprintf(os.Stderr, "Count: %d\r", cnt)
		}
		if _, ok := csvMap[line.key]; ok {
			csvMap[line.key][line.index] = line.value
		} else {
			s := make([]string, len(args))
			s[line.index] = line.value
			csvMap[line.key] = s
		}
	}

	if len(csvMap) == 0 {
		fmt.Println("There is no output !")
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
	writer.Comma = ','
	writer.UseCRLF = true

	// Write header.
	err = writer.Write(append([]string{keyName}, args...))
	if err != nil {
		log.Fatalln(err)
	}

	// map to array.
	var csvArray records
	for k, v := range csvMap {
		csvArray = append(csvArray, append([]string{k}, v...))
	}

	// sort
	if sorts == "" {
		sorts = "0"
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
