// Copyright © 2016 yukimemi <yukimemi@gmail>
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
	"github.com/spf13/viper"
	"github.com/yukimemi/file"
)

// Cmd options.
var cfgFile string
var out string
var sortFlg bool

type cmdInfo struct {
	cmdFile string
	cmdDir  string
	cmdName string
}

// FileInfo is file infomation.
type FileInfo struct {
	Full string    `json:"full"`
	Abs  string    `json:"abs"`
	Name string    `json:"name"`
	Time time.Time `json:"time"`
	Size string    `json:"size"`
	Mode string    `json:"mode"`
}

// FileInfos is FileInfo slice.
type FileInfos []FileInfo

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "gfi",
	Short: "Get file information",
	Long: `Get file information command. filepath, size, mode etc.
examples and usage of using your application. For example:

	gfi path/to/dir

`,

	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		// Get cmd info.
		cmdFile, err := filepath.Abs(os.Args[0])
		if err != nil {
			_ = fmt.Errorf("Error get cmd file name")
			return
		}
		cmdDir := filepath.Dir(cmdFile)
		cmdInfo := cmdInfo{
			cmdFile: cmdFile,
			cmdDir:  cmdDir,
			cmdName: file.BaseName(cmdFile),
		}

		fis := make(FileInfos, 0)
		wg := new(sync.WaitGroup)
		for _, root := range args {
			wg.Add(1)
			go func(root string) {
				defer wg.Done()
				fi, err := getFileInfo(root)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error occur when get [%s] directory file information.\n", root)
				} else {
					fis = append(fis, fi...)
				}
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
			return filepath.Join(cmdInfo.cmdDir, now.Format("20060102-150405.000")+".json")
		}()
		j, err := json.MarshalIndent(fis, "", "\t")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occur at MarshalIndent for [%v]", fis)
		}
		file, err := os.Create(jsonPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error occur at create [%s] json file.\n", jsonPath)
		}
		defer file.Close()
		n, err := file.Write(j)
		fmt.Printf("Write to [%s] file. ([%d] bytes)", jsonPath, n)

	},
}

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

	// Output json path.
	RootCmd.PersistentFlags().StringVarP(&out, "out", "o", "", "Json output path")
	// Sort with fullpath for josn.
	RootCmd.PersistentFlags().BoolVarP(&sortFlg, "sort", "s", false, "Sort flag")

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

func getFileInfo(root string) (FileInfos, error) {

	var fis FileInfos
	opt := file.Option{
		Recurse: true,
	}

	fs, err := file.GetFilesAndDirs(root, opt)
	if err != nil {
		return nil, err
	}

	for f := range fs {
		if f.Err != nil {
			return nil, f.Err
		}
		info := &FileInfo{
			Full: f.Path,
			Abs:  file.ShareToAbs(f.Path),
			Name: f.Fi.Name(),
			Time: f.Fi.ModTime(),
			Size: fmt.Sprint(f.Fi.Size()),
			Mode: f.Fi.Mode().String(),
		}
		fis = append(fis, *info)
	}

	return fis, err
}

// Len returns FileInfos length.
func (f FileInfos) Len() int {
	return len(f)
}

// Less returns which FileInfo is less.
func (f FileInfos) Less(i, j int) bool {
	return f[i].Abs < f[j].Abs
}

// Swap is FileInfos swap func.
func (f FileInfos) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
