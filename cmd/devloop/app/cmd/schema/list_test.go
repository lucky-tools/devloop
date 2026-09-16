/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package schema

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/testutil"
)

func TestListPlain(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var out bytes.Buffer

		err := list(&out, "plain")
		t.CheckNoError(err)

		t.CheckDeepEqual(latest.Version+"\n", out.String())
	})
}

func TestListJson(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var out bytes.Buffer

		err := list(&out, "json")
		t.CheckNoError(err)

		t.CheckDeepEqual(fmt.Sprintf("{\"versions\":[\"%s\"]}\n", latest.Version), out.String())
	})
}

func TestListInvalidType(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		var out bytes.Buffer

		err := list(&out, "invalid")
		t.CheckErrorContains(`invalid output type: "invalid". Must be "plain" or "json"`, err)
	})
}
