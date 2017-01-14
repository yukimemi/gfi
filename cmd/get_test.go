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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGetCmdRun is test getCmd.Run.
func TestGetCmdRun(t *testing.T) {

	var (
		ei, ai int
		es, as string
		err    error
		fis    Output
	)

	tmp := setup()
	defer shutdown(tmp)

	j1 := filepath.Join(tmp, json1)
	RootCmd.SetArgs([]string{"get", "-s", "-o", j1, tmp})
	err = RootCmd.Execute()
	if err != nil {
		t.FailNow()
	}

	// Check json.
	f1, err := os.Open(j1)
	if err != nil {
		t.FailNow()
	}

	jr1 := json.NewDecoder(f1)
	err = jr1.Decode(&fis)

	if err != nil {
		t.FailNow()
	}

	// Check count.
	ei = fileCnt*2 + dirCnt
	ai = fis.Count

	if ai != ei {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", ei, ai)
	}

	// Check FileInfos.
	index := 0
	// Index 0.
	es = "dir0"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 1.
	index++
	es = "file0"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 2.
	index++
	es = "dir1"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 3.
	index++
	es = "file1"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 4.
	index++
	es = "dir2"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 5.
	index++
	es = "file2"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 6.
	index++
	es = "file0"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 7.
	index++
	es = "file1"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	// Index 8.
	index++
	es = "file2"
	as = fis.FileInfos[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

}
