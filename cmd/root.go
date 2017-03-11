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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yukimemi/core"
)

// FileInfoValue is FileInfo type.
type FileInfoValue int

// DirInfoValue is DirInfo type.
type DirInfoValue int

const (
	// FILE is file.
	FILE = "file"
	// DIR is directory.
	DIR = "directory"
	// COUNT is file count.
	COUNT = "Count"
	// UTF8 is csv encoding.
	UTF8 = "utf8"
	// SJIS is csv encoding.
	SJIS = "sjis"
)

const (
	// FileFull is full path
	FileFull FileInfoValue = iota + 1
	// FileRel is relative path.
	FileRel
	// FileAbs is absolute path.
	FileAbs
	// FileName is file name.
	FileName
	// FileTime is file modified time.
	FileTime
	// FileSize is file size.
	FileSize
	// FileMode is file permissions.
	FileMode
	// FileType is file or directory.
	FileType
	// FileMax is Max
	FileMax = iota
)

const (
	// DirFull is full path
	DirFull DirInfoValue = iota + 1
	// DirRel is relative path.
	DirRel
	// DirAbs is absolute path.
	DirAbs
	// DirName is directory name.
	DirName
	// DirTime is directory modified time.
	DirTime
	// DirSize is directory size.
	DirSize
	// DirFileCount is file count.
	DirFileCount
	// DirDirCount is directory count or directory.
	DirDirCount
	// DirMax is Max
	DirMax = iota
)

var (
	// Cmd options.
	cfgFile  string
	out      string
	fileOnly bool
	dirOnly  bool
	sjisOut  bool
	matches  []string
	ignores  []string
	sorts    string
	errSkip  bool
	silent   bool
	// Other variables.
	err error
	ci  core.Cmd
	cnt = 0
)

// FileInfo is file infomation.
type FileInfo struct {
	Full string
	Rel  string
	Abs  string
	Name string
	Time string
	Size string
	Mode string
	Type string
}

// DirInfo is file infomation.
type DirInfo struct {
	Full      string
	Rel       string
	Abs       string
	Name      string
	Time      string
	Size      string
	FileCount int64
	DirCount  int64
}

// FileInfos is FileInfo slice.
type FileInfos []FileInfo

// DirInfos is DirInfo slice.
type DirInfos []DirInfo

type records [][]string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{Use: "gfi"}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.gfi.yaml)")

	// Get pwd.
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatalln(err)
	}

	// Output csv path.
	csvPath := filepath.Join(pwd, time.Now().Format("20060102-150405.000")+".csv")
	RootCmd.PersistentFlags().StringVarP(&out, "out", "o", csvPath, "Csv output path")
	// Verbose flag.
	RootCmd.PersistentFlags().BoolVarP(&silent, "silent", "S", false, "Print no count")
	// File only flag.
	RootCmd.PersistentFlags().BoolVarP(&fileOnly, "file", "f", false, "Get information file only")
	// Directory only flag.
	RootCmd.PersistentFlags().BoolVarP(&dirOnly, "dir", "d", false, "Get information directory only")
	// Whether output csv in ShiftJIS encoding.
	RootCmd.PersistentFlags().BoolVarP(&sjisOut, "sjisout", "j", false, "Output csv in ShiftJIS encoding")
	// Matches list.
	RootCmd.PersistentFlags().StringArrayVarP(&matches, "match", "m", nil, "Match list (Regexp)")
	// Ignores list.
	RootCmd.PersistentFlags().StringArrayVarP(&ignores, "ignore", "i", nil, "Ignore list (Regexp)")

	// log setting.
	log.SetFlags(log.Lshortfile)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".gfi")  // name of config file (without extension)
	viper.AddConfigPath("$HOME") // adding home directory as first search path
	viper.AutomaticEnv()         // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// Len returns FileInfos length.
func (f FileInfos) Len() int {
	return len(f)
}

// Less returns which FileInfo is less.
func (f FileInfos) Less(i, j int) bool {
	return f[i].Full < f[j].Full
}

// Swap is FileInfos swap func.
func (f FileInfos) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

// Len returns DirInfos length.
func (d DirInfos) Len() int {
	return len(d)
}

// Less returns which DirInso is less.
func (d DirInfos) Less(i, j int) bool {
	return d[i].Full < d[j].Full
}

// Swap is DirInfos swap func.
func (d DirInfos) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
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

func (fiv FileInfoValue) String() string {
	switch fiv {
	case FileFull:
		return "Full"
	case FileRel:
		return "Rel"
	case FileAbs:
		return "Abs"
	case FileName:
		return "Name"
	case FileTime:
		return "Time"
	case FileSize:
		return "Size"
	case FileMode:
		return "Mode"
	case FileType:
		return "Type"
	}
	return ""
}

func (div DirInfoValue) String() string {
	switch div {
	case DirFull:
		return "Full"
	case DirRel:
		return "Rel"
	case DirAbs:
		return "Abs"
	case DirName:
		return "Name"
	case DirTime:
		return "Time"
	case DirSize:
		return "Size"
	case DirFileCount:
		return "FileCount"
	case DirDirCount:
		return "DirCount"
	}
	return ""
}
