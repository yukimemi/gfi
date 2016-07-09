package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
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

var wg sync.WaitGroup
var semaphore = make(chan int, runtime.NumCPU())

func run(args []string) int {

	var err error

	// Show cpu num.
	Log.Infof("CPU NUM: [%d]", runtime.NumCPU())

	// Ready csv writer.
	now := time.Now()
	file, err := os.Create(now.Format("20060102150405006") + ".csv")
	FailOnError(err)
	defer file.Close()

	writer := csv.NewWriter(transform.NewWriter(file, japanese.ShiftJIS.NewEncoder()))
	writer.UseCRLF = true
	writer.Comma = '\t'

	record := make(chan []string)

	// Walk path and get file info.
	for _, root := range args {
		wg.Add(1)
		go func(root string, record chan []string) {
			defer wg.Done()
			semaphore <- 1
			getInfo(root, record)
			<-semaphore
		}(root, record)
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
	// Log.Infof("path: [%s]\n", root)

	infos, err := ioutil.ReadDir(root)
	WarnOnError(err)

	for _, fi := range infos {
		full := filepath.Join(root, fi.Name())
		if fi.IsDir() {
			wg.Add(1)
			go func(root string, record chan []string) {
				defer wg.Done()
				semaphore <- 1
				getInfo(root, record)
				<-semaphore
			}(full, record)
		} else {
			record <- []string{full, fi.Name(), fi.ModTime().Format("2006/01/02 15:04:05.006"), fmt.Sprint(fi.Size()), fi.Mode().String()}
		}
	}

}
