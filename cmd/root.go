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
	// Count is file count.
	Count FileInfoValue = iota + 1
	// Full is full path
	Full
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
)

// Cmd options.
var (
	cfgFile  string
	out      string
	fileOnly bool
	dirOnly  bool
	matches  []string
	ignores  []string
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
	Full string    `json:"full"`
	Rel  string    `json:"rel"`
	Abs  string    `json:"abs"`
	Name string    `json:"name"`
	Time time.Time `json:"time"`
	Size string    `json:"size"`
	Mode string    `json:"mode"`
	Type string    `json:"type"`
}

// Output is output json struct.
type Output struct {
	Count     int `json:"count"`
	FileInfos `json:"fileinfos"`
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
	// Output json or csv path.
	RootCmd.PersistentFlags().StringVarP(&out, "out", "o", "", "Json/Csv output path")
	// File only flag.
	RootCmd.PersistentFlags().BoolVarP(&fileOnly, "file", "f", false, "Get information file only")
	// Directory only flag.
	RootCmd.PersistentFlags().BoolVarP(&dirOnly, "dir", "d", false, "Get information directory only")
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
	fmt.Printf("CmdFile : [%s]\n", ci.File)
	fmt.Printf("CmdDir  : [%s]\n", ci.Dir)
	fmt.Printf("CmdName : [%s]\n", ci.Name)
	fmt.Printf("Cwd     : [%s]\n", ci.Cwd)
	return ci, nil

}
