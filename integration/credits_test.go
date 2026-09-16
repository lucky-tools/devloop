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

package integration

import (
	"os"
	"testing"

	"github.com/lucky-tools/devloop/integration/devloop"
	"github.com/lucky-tools/devloop/testutil"
)

func TestCredits(t *testing.T) {
	testutil.Run(t, "credits", func(t *testutil.T) {
		MarkIntegrationTest(t.T, CanRunWithoutGcp)

		tmpDir := t.NewTempDir().Chdir()

		out, err := devloop.Credits().RunWithCombinedOutput(t.T)
		t.CheckNoError(err)
		t.CheckContains("Successfully exported third party notices", string(out))

		content, err := os.ReadFile(tmpDir.Path("devloop-credits/github.com/docker/docker/LICENSE"))
		t.CheckNoError(err)
		t.CheckContains("Apache License", string(content))
	})
}

func TestCreditsDir(t *testing.T) {
	testutil.Run(t, "credits", func(t *testutil.T) {
		MarkIntegrationTest(t.T, CanRunWithoutGcp)

		tmpDir := t.NewTempDir().Chdir()
		tmpDir.Mkdir("test/devloop-credits")

		out, err := devloop.Credits("-d", "test/devloop-credits/credits").RunWithCombinedOutput(t.T)
		t.CheckNoError(err)
		t.CheckContains("Successfully exported third party notices", string(out))

		content, err := os.ReadFile(tmpDir.Path("test/devloop-credits/credits/github.com/docker/docker/LICENSE"))
		t.CheckNoError(err)
		t.CheckContains("Apache License", string(content))
	})
}
