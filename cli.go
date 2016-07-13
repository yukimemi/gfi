package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    int = 0
	ExitCodeError int = 1 + iota
)

// CLI is the command line object
type CLI struct {
	log       *logrus.Logger
	wg        sync.WaitGroup
	semaphore chan int
	gfis      []*gfi
	cmdFile   string
	cmdDir    string
}

// gfi is a tool for get file information
type gfi struct {
	ip   string
	root string
	path string
	fi   []map[string]string
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		verbose bool
		ip      string
		version bool
		err     error
	)

	// Log
	if cli.log == nil {
		fmt.Println("No Logger set. New logrus.")
		cli.log = logrus.New()
	}

	// Set semaphore
	if cli.semaphore == nil {
		cli.log.Infof("No semaphore set. Set semaphore to [%d]", runtime.NumCPU())
		cli.semaphore = make(chan int, runtime.NumCPU())
	}

	// Define option flag parse
	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	flags.BoolVar(&verbose, "verbose", false, "Show output")
	flags.BoolVar(&verbose, "v", false, "Same verbose. (Short option)")

	flags.StringVar(&ip, "ip", "", "Set ip if want to get remote file information.")
	flags.StringVar(&ip, "i", "", "Same ip. (Short option)")

	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	// Parse commandline flag
	if err = flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version
	if version {
		fmt.Fprintf(os.Stderr, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	// Show output
	if verbose {
		cli.log.Info("Set verbose mode. Output log to Stdout.")
		cli.log.Out = os.Stdout
	} else {
		cli.log.Out = ioutil.Discard
	}

	// Set log level.
	cli.log.Level = logrus.InfoLevel

	// Show cpu num
	cli.log.Infof("CPU NUM: [%d]", runtime.NumCPU())

	// Get cmd info.
	cli.cmdFile, err = filepath.Abs(os.Args[0])
	cli.failOnError(err)
	cli.cmdDir = filepath.Dir(cli.cmdFile)
	cli.log.Infof("cmdFile: [%s] cmdDir: [%s]", cli.cmdFile, cli.cmdDir)

	// Walk path and get file info.
	for _, root := range flags.Args() {

		// Get file information and write csv under the root.
		cli.wg.Add(1)
		go func(root string, ip string) {
			defer cli.wg.Done()
			cli.getInfo(root, ip)
		}(root, ip)

		cli.wg.Wait()
	}

	return ExitCodeOK
}

func (cli *CLI) getInfo(root string, ip string) {

	var (
		err  error
		base string
		q    chan map[string]string
	)

	if ip != "" {
		base = filepath.Join("\\\\"+ip, root)
		base = strings.Replace(base, ":", "$", -1)
	} else {
		base = root
	}

	// Check base exist.
	fi, err := os.Stat(base)
	cli.failOnError(err)
	if !fi.IsDir() {
		cli.log.Fatalf("[%s] is not a directory.", base)
	}

	// Get file information.
	q = func(base string) chan map[string]string {
		q := make(chan map[string]string)
		wg := new(sync.WaitGroup)

		cli.log.Infof("base: [%s]", base)

		var fn func(p string)
		fn = func(p string) {
			cli.semaphore <- 1
			defer func() {
				wg.Done()
				<-cli.semaphore
			}()

			fis, err := ioutil.ReadDir(p)
			if err != nil {
				cli.warnOnError(err)
				return
			}

			for _, fi := range fis {
				if fi.IsDir() {
					wg.Add(1)
					go fn(filepath.Join(p, fi.Name()))
				} else {
					format := "2006/01/02 15:04:05.006"
					full, err := filepath.Abs(filepath.Join(p, fi.Name()))
					cli.failOnError(err)
					q <- map[string]string{
						"full": full,
						"name": fi.Name(),
						"time": fi.ModTime().Format(format),
						"size": fmt.Sprint(fi.Size()),
						"mode": fi.Mode().String(),
					}
				}
			}
		}

		wg.Add(1)
		go fn(base)

		// Wait.
		go func() {
			wg.Wait()
			close(q)
		}()

		return q
	}(base)

	// Write to csv.
	now := time.Now()
	outPath := filepath.Join(cli.cmdDir, filepath.Base(base))
	outName := now.Format("20060102-150405006") + ".csv"
	cli.mkdir(outPath)
	// Ready csv writer.
	file, err := os.Create(filepath.Join(outPath, outName))
	cli.failOnError(err)
	defer file.Close()

	writer := csv.NewWriter(transform.NewWriter(file, japanese.ShiftJIS.NewEncoder()))
	writer.UseCRLF = true
	writer.Comma = '\t'

	// Receive file information.
	for s := range q {
		cli.log.Debug(s)
		writer.Write([]string{s["full"], s["name"], s["time"], s["size"], s["mode"]})
	}
	writer.Flush()
}

// failOnError is easy to judge error.
func (cli *CLI) failOnError(e error) {
	if e != nil {
		cli.log.Fatal(e.Error())
	}
}

// warnOnError is easy to judge warn.
func (cli *CLI) warnOnError(e error) {
	if e != nil {
		cli.log.Warn(e.Error())
	}
}

// mkdir is make directory
func (cli *CLI) mkdir(dir string) error {
	d, e := os.Stat(dir)
	if e != nil {
		return os.Mkdir(dir, os.ModePerm)
	} else if d.IsDir() {
		cli.log.Info("[%s] is already exists !", dir)
		return nil
	}

	return errors.New(fmt.Sprintf("Can't create [%s] directory !", dir))
}
