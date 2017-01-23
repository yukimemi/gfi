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
	"encoding/csv"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDiffCmdRun is test diffCmd.Run.
func TestDiffCmdRun(t *testing.T) {

	var (
		err error
	)

	tmp := setup()
	t.Log(tmp)
	/* defer shutdown(tmp) */

	c1 := filepath.Join(tmp, getCsv1)
	RootCmd.SetArgs([]string{"get", "-s", "-o", c1, tmp})
	err = RootCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// Change size and time.
	err = ioutil.WriteFile(filepath.Join(tmp, "file0"), []byte{'t'}, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Chtimes(filepath.Join(tmp, "file0"), time.Now(), time.Now().Add(time.Minute*5))
	if err != nil {
		t.Fatal(err)
	}

	c2 := filepath.Join(tmp, getCsv2)
	RootCmd.SetArgs([]string{"get", "-s", "-o", c2, tmp})
	err = RootCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	dc1 := filepath.Join(tmp, diffCsv1)
	RootCmd.SetArgs([]string{"diff", "-o", dc1, c1, c2})
	err = RootCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// Check csv.
	f, err := os.Open(dc1)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	reader := csv.NewReader(f)
	reader.Comma = '\t'
	cnt := 0
	for {
		r, err := reader.Read()
		if err == io.EOF {
			break
		} else {
			switch cnt {
			case 0:
				if r[0] != "Path" {
					t.Fatalf("Expect: [Path] Actual: [%v]", r[0])
				}
				if r[1] != "Type" {
					t.Fatalf("Expect: [Type] Actual: [%v]", r[1])
				}
				if r[2] != "Diff" {
					t.Fatalf("Expect: [Diff] Actual: [%v]", r[2])
				}
			case 1:
				if filepath.Base(r[0]) != filepath.Base(tmp) {
					t.Fatalf("Expect: [%v] Actual: [%v]", filepath.Base(tmp), r[0])
				}
				if r[2] != FileTime.String() {
					t.Fatalf("Expect: [%v] Actual: [%v]", FileTime.String(), r[2])
				}
			case 3:
				if filepath.Base(r[0]) != "file0" {
					t.Fatalf("Expect: [file0] Actual: [%v]", r[0])
				}
				if r[2] != FileSize.String() {
					t.Fatalf("Expect: [%v] Actual: [%v]", FileSize.String(), r[2])
				}
			case 2:
				if filepath.Base(r[0]) != "file0" {
					t.Fatalf("Expect: [file0] Actual: [%v]", r[0])
				}
				if r[2] != FileTime.String() {
					t.Fatalf("Expect: [%v] Actual: [%v]", FileTime.String(), r[2])
				}
			case 5:
				if r[0] != c1 {
					t.Fatalf("Expect: [%v] Actual: [%v]", c1, r[0])
				}
				if r[2] != FileSize.String() {
					t.Fatalf("Expect: [%v] Actual: [%v]", FileSize.String(), r[2])
				}
			case 4:
				if r[0] != c1 {
					t.Fatalf("Expect: [%v] Actual: [%v]", c1, r[0])
				}
				if r[2] != FileTime.String() {
					t.Fatalf("Expect: [%v] Actual: [%v]", FileTime.String(), r[2])
				}
			case 6:
				if r[0] != c2 {
					t.Fatalf("Expect: [%v] Actual: [%v]", c2, r[0])
				}
				if r[2] != FileFull.String() {
					t.Fatalf("Expect: [%v] Actual: [%v]", FileFull.String(), r[2])
				}
			}
		}
		cnt++
	}

}
