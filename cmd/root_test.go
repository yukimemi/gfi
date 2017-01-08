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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const (
	fileCnt = 3
	dirCnt  = 3
	json1   = "one.json"
	json2   = "two.json"
	json3   = "three.json"
	csv1    = "diff.csv"
)

func setup() string {
	temp, err := ioutil.TempDir("", "test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for i := 0; i < fileCnt; i++ {
		f, _ := os.Create(filepath.Join(temp, "file"+fmt.Sprint(i)))
		f.Close()
	}
	for i := 0; i < dirCnt; i++ {
		d := filepath.Join(temp, "dir"+fmt.Sprint(i))
		err := os.MkdirAll(d, os.ModePerm)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		f, _ := os.Create(filepath.Join(d, "file"+fmt.Sprint(i)))
		f.Close()
	}
	return temp
}

func shutdown(temp string) {
	os.RemoveAll(temp)
}

// TestMain is entry point.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
