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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yukimemi/file"
)

const fileCnt = 3
const dirCnt = 3

func setup() {
	pwd, _ := os.Getwd()
	test := filepath.Join(pwd, "test")
	os.MkdirAll(test, os.ModePerm)
	for i := 0; i < fileCnt; i++ {
		os.Create(filepath.Join(test, "file"+fmt.Sprint(i)))
	}
	for i := 0; i < dirCnt; i++ {
		d := filepath.Join(test, "dir"+fmt.Sprint(i))
		os.MkdirAll(d, os.ModePerm)
		os.Create(filepath.Join(d, "file"+fmt.Sprint(i)))
	}
}

func shutdown() {
	pwd, _ := os.Getwd()
	test := filepath.Join(pwd, "test")
	if file.IsExistDir(test) {
		os.RemoveAll(test)
	}
}

// TestExecute is test getFileInfo func.
func TestExecute(t *testing.T) {
	setup()
	pwd, _ := os.Getwd()
	test := filepath.Join(pwd, "test")

	_, e := getFileInfo(test)
	if e != nil {
		t.FailNow()
	}
	shutdown()
}

// TestMain is entry point.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
