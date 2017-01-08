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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDiffCmdRun is test diffCmd.Run.
func TestDiffCmdRun(t *testing.T) {

	var (
		// ei, ai int
		// es, as string
		err error
		// fis    Output
	)

	tmp := setup()

	j1 := filepath.Join(tmp, json1)
	RootCmd.SetArgs([]string{"get", "-s", "-o", j1, tmp})
	err = RootCmd.Execute()
	if err != nil {
		t.FailNow()
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

	j2 := filepath.Join(tmp, json2)
	RootCmd.SetArgs([]string{"get", "-s", "-o", j2, tmp})
	err = RootCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

	c1 := filepath.Join(tmp, csv1)
	RootCmd.SetArgs([]string{"diff", "-o", c1, j1, j2})
	err = RootCmd.Execute()
	if err != nil {
		t.Fatal(err)
	}

}
