package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var cmdDef = &Command{
	UsageLine: "gfi is get file information tool",
	Short:     "Get file info",
	Long:      "Get file information.",
	Run:       run,
}

func worker(num int) chan<- func() {
	tasks := make(chan func())
	for i := 0; i < num; i++ {
		go func() {
			for f := range tasks {
				chNum <- 1
				f()
				chNum <- -1
			}
		}()
	}
	return tasks
}

// for test
var chNum = make(chan int)
var runNum = 0

func showRunNum() {
	for num := range chNum {
		runNum += num
		Log.Infof("Run num: [%3d]", runNum)
	}
}

func run(args []string) int {

	var err error
	var wg sync.WaitGroup
	tasks := worker(3)

	// Ready csv writer.
	now := time.Now()
	file, err := os.Create(now.Format("20060102150405006") + ".csv")
	FailOnError(err)
	defer file.Close()

	writer := csv.NewWriter(transform.NewWriter(file, japanese.ShiftJIS.NewEncoder()))
	writer.UseCRLF = true
	writer.Comma = '\t'

	record := make(chan []string)

	// test
	go showRunNum()

	// Walk path and get file info.
	for _, root := range args {
		wg.Add(1)
		tasks <- func() {
			func(root string, record chan []string) {
				defer wg.Done()
				getInfo(root, record)
			}(root, record)
		}
	}

	// Get records.
	go func() {
		for r := range record {
			writer.Write(r)
		}
	}()
	wg.Wait()
	close(record)

	writer.Flush()

	return 0
}

func getInfo(root string, record chan []string) {

	var err error
	var wg sync.WaitGroup
	tasks := worker(3)
	// Log.Infof("path: [%s]\n", root)

	infos, err := ioutil.ReadDir(root)
	WarnOnError(err)

	for _, fi := range infos {
		full := filepath.Join(root, fi.Name())
		if fi.IsDir() {
			wg.Add(1)
			tasks <- func() {
				func(root string, record chan []string) {
					defer wg.Done()
					getInfo(root, record)
				}(full, record)
			}
		} else {
			record <- []string{full, fi.Name(), fi.ModTime().Format("2006/01/02 15:04:05.006"), fmt.Sprint(fi.Size()), fi.Mode().String()}
		}
	}

	wg.Wait()

}
