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
	"os"
	"path/filepath"
	"testing"
)

// TestGetCmdRun is test getCmd.Run.
func TestGetCmdRun(t *testing.T) {

	var (
		es, as string
		err    error
	)

	tmp := setup()
	t.Log(tmp)
	defer shutdown(tmp)

	c1 := filepath.Join(tmp, getCsv1)
	RootCmd.SetArgs([]string{"get", "-s", "-o", c1, tmp})
	err = RootCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	// Check csv.
	f1, err := os.Open(c1)
	if err != nil {
		t.Fatal(err)
	}

	reader := csv.NewReader(f1)
	reader.Comma = ','
	// Skip header.
	_, err = reader.Read()
	if err != nil {
		t.Fatal(err)
	}
	left, err := reader.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// Csv to FileInfos.
	fis := make(FileInfos, 0)
	for _, r := range left {
		fis = append(fis, *csvToFileInfo(r))
	}

	// Check FileInfos.
	index := 0
	es = filepath.Base(tmp)
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "dir0"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "file0"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "dir1"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "file1"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "dir2"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "file2"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "file0"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "file1"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

	index++
	es = "file2"
	as = fis[index].Name
	if as != es {
		t.Fatalf("Expected: [%v] but actual: [%v]\n", es, as)
	}

}
