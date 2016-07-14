package main

import (
	"encoding/csv"
	"encoding/json"
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

// CLI is the command line object.
type CLI struct {
	log       *logrus.Logger
	wg        sync.WaitGroup
	semaphore chan int
	cmdFile   string
	cmdDir    string
	ip        string
	jsn       bool
	csv       bool
	out       string
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	var (
		verbose bool
		level   string
		version bool
		err     error
	)

	// Get cmd info.
	cli.cmdFile, err = filepath.Abs(os.Args[0])
	cli.failOnError(err)
	cli.cmdDir = filepath.Dir(cli.cmdFile)

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

	flags.StringVar(&cli.ip, "i", "", "Set ip if you want to get remote file information.")
	flags.BoolVar(&cli.jsn, "j", false, "Output to json.")
	flags.BoolVar(&cli.csv, "c", false, "Output to csv.")
	flags.StringVar(&cli.out, "o", cli.cmdDir, "Output path.")
	flags.StringVar(&level, "l", "Warn", "Set log level. (Debug, Info, Warn, Error, Fatal, Panic)")
	flags.BoolVar(&verbose, "verbose", false, "Show output.")
	flags.BoolVar(&version, "version", false, "Print version information and quit.")

	// Parse commandline flag
	if err = flags.Parse(args[1:]); err != nil {
		return ExitCodeError
	}

	// Show version.
	if version {
		fmt.Fprintf(os.Stderr, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	// Check arg.
	if flags.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flags.PrintDefaults()
		return ExitCodeError
	}

	// Show output.
	if verbose {
		cli.log.Info("Set verbose mode. Output log to Stdout.")
		cli.log.Out = os.Stdout
	} else {
		cli.log.Out = ioutil.Discard
	}

	// Set output type.
	if !cli.jsn && !cli.csv {
		cli.jsn = true
	}

	// Set log level.
	if level != "Warn" {
		cli.log.Infof("Set log level to [%s]", level)
	}
	switch level {
	case "Debug":
		cli.log.Level = logrus.DebugLevel
	case "Info":
		cli.log.Level = logrus.InfoLevel
	case "Warn":
		cli.log.Level = logrus.WarnLevel
	case "Error":
		cli.log.Level = logrus.ErrorLevel
	case "Fatal":
		cli.log.Level = logrus.FatalLevel
	case "Panic":
		cli.log.Level = logrus.PanicLevel
	}

	// Show cmd info and cpu num.
	cli.log.Infof("cmdFile: [%s] cmdDir: [%s]", cli.cmdFile, cli.cmdDir)
	cli.log.Infof("CPU NUM: [%d]", runtime.NumCPU())

	// Walk path and get file info.
	for _, base := range flags.Args() {

		// Get file information and write csv under the base.
		cli.wg.Add(1)
		go func(base string, ip string) {
			defer cli.wg.Done()
			cli.getInfo(base, ip)
		}(base, cli.ip)

		cli.wg.Wait()
	}

	return ExitCodeOK
}

func (cli *CLI) getInfo(base string, ip string) {

	var (
		err error
		q   chan map[string]string
	)

	if ip != "" {
		base = filepath.Join("\\\\"+ip, base)
		base = strings.Replace(base, ":", "$", -1)
	} else {
		base = base
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

	// Write to output.
	now := time.Now()
	outPath := filepath.Join(cli.out, base)
	cli.mkdir(outPath)
	var outName string
	if cli.csv {
		outName = now.Format("20060102-150405.000") + ".csv"
	} else if cli.jsn {
		outName = now.Format("20060102-150405.000") + ".json"
	}

	// Ready writer.
	file, err := os.Create(filepath.Join(outPath, outName))
	cli.failOnError(err)
	defer file.Close()

	var csvWriter *csv.Writer
	if cli.csv {
		csvWriter = csv.NewWriter(transform.NewWriter(file, japanese.ShiftJIS.NewEncoder()))
		csvWriter.UseCRLF = true
		csvWriter.Comma = '\t'
	}

	// Receive file information.
	is := make([]map[string]string, 0)
	for s := range q {
		cli.log.Debug(s)
		if cli.csv {
			csvWriter.Write([]string{s["full"], s["name"], s["time"], s["size"], s["mode"]})
		} else if cli.jsn {
			is = append(is, s)
		}
	}
	if cli.csv {
		csvWriter.Flush()
	}

	// Write to json.
	if cli.jsn {
		j, err := json.Marshal(is)
		cli.failOnError(err)
		n, err := file.Write(j)
		cli.log.Infof("Write to [%s] file. ([%d] bytes)", filepath.Join(outPath, outName), n)
	}
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
		cli.log.Infof("[%s] is already exists !", dir)
		return nil
	}

	return errors.New(fmt.Sprintf("Can't create [%s] directory !", dir))
}
