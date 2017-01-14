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
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yukimemi/file"
)

// FileInfoValue is FileInfo type.
type FileInfoValue int

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
	// Full is full path
	Full FileInfoValue = iota + 1
	// Rel is relative path.
	Rel
	// Abs is absolute path.
	Abs
	// Name is file name.
	Name
	// Time is file modified time.
	Time
	// Size is file size.
	Size
	// Mode is file permissions.
	Mode
	// Type is file or directory.
	Type
	// Max is Max
	Max = iota
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
	// Other variables.
	err error
	ci  Cmd
	cnt = 0
)

// Cmd is command infomation.
type Cmd struct {
	File string
	Dir  string
	Name string
	Cwd  string
}

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

// FileInfos is FileInfo slice.
type FileInfos []FileInfo

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

	// Get command info.
	ci, err = GetCmdInfo()
	if err != nil {
		log.Fatalln(err)
	}

	// Output csv path.
	csvPath := filepath.Join(ci.Cwd, time.Now().Format("20060102-150405.000")+".csv")
	RootCmd.PersistentFlags().StringVarP(&out, "out", "o", csvPath, "Csv output path")
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

// GetCmdInfo return struct of Cmd.
func GetCmdInfo() (Cmd, error) {

	var (
		ci  Cmd
		err error
	)

	// Get cmd info.
	cmdFile, err := file.GetCmdPath(os.Args[0])
	if err != nil {
		return ci, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ci, err
	}
	cmdDir := filepath.Dir(cmdFile)
	ci = Cmd{
		File: cmdFile,
		Dir:  cmdDir,
		Name: file.BaseName(cmdFile),
		Cwd:  cwd,
	}
	return ci, nil

}

func getGlobArgs(args []string) ([]string, error) {

	var (
		err error
		a   = make([]string, 0)
	)
	for _, v := range args {
		files, err := filepath.Glob(v)
		if err != nil {
			log.Println(err)
			return nil, err
		}
		a = append(a, files...)
	}

	return a, err
}

func (fiv FileInfoValue) String() string {
	switch fiv {
	case Full:
		return "Full"
	case Rel:
		return "Rel"
	case Abs:
		return "Abs"
	case Name:
		return "Name"
	case Time:
		return "Time"
	case Size:
		return "Size"
	case Mode:
		return "Mode"
	case Type:
		return "Type"
	}
	return ""
}
