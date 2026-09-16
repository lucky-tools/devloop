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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/lucky-tools/devloop/integration/devloop"
	"github.com/lucky-tools/devloop/testutil"
)

func TestDiagnoseExamples(t *testing.T) {
	examples, err := folders("examples")
	failNowIfError(t, err)
	if len(examples) == 0 {
		t.Fatal("didn't find any example")
	}

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			MarkIntegrationTest(t, CanRunWithoutGcp)
			dir := filepath.Join("examples", example)

			if _, err := os.Stat(filepath.Join(dir, "devloop.yaml")); os.IsNotExist(err) {
				t.Skip("skipping diagnose in " + dir)
			}

			devloop.Diagnose().InDir(dir).RunOrFail(t)
		})
	}
}

func TestDiagnose(t *testing.T) {
	tests := []struct {
		description string
		dir         string
		outputFile  string
		args        []string
		envs        map[string]string
	}{
		{
			description: "single devloop.yaml outside of source dir",
			dir:         "testdata/diagnose/temp-config",
			outputFile:  "abc.txt",
		},
		{
			description: "apply replacements to templates in devloop.yaml",
			dir:         "testdata/diagnose/direct-templates",
			outputFile:  "abc.txt",
			args:        []string{"--enable-templating"},
			envs:        map[string]string{"AAA": "aaa"},
		},
	}

	for _, test := range tests {
		MarkIntegrationTest(t, CanRunWithoutGcp)
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.envs != nil {
				for k, v := range test.envs {
					t.Setenv(k, v)
				}
			}
			tmpDir := testutil.NewTempDir(t.T)
			configContents, err := os.ReadFile(filepath.Join(test.dir, "devloop.yaml"))
			t.CheckNoError(err)
			templ, err := os.ReadFile(filepath.Join(test.dir, "diagnose.tmpl"))
			t.CheckNoError(err)
			tmpDir.Write("devloop.yaml", string(configContents))
			args := []string{"--yaml-only", "--output", tmpDir.Path(test.outputFile), "-f", tmpDir.Path("devloop.yaml")}
			args = append(args, test.args...)
			devloop.Diagnose(args...).
				InDir(test.dir).RunOrFail(t.T)
			outTemplate := template.Must(template.New("tmpl").Parse(string(templ)))
			cwd, err := filepath.Abs(test.dir)
			t.CheckNoError(err)
			expected := &bytes.Buffer{}
			outTemplate.Execute(expected, map[string]string{"Root": cwd})

			outputPath := tmpDir.Path(test.outputFile)
			out, err := os.ReadFile(outputPath)
			t.CheckNoError(err)
			t.CheckDeepEqual(expected.String(), string(out), testutil.YamlObj(t.T))
		})
	}
}

func folders(root string) ([]string, error) {
	var folders []string

	files, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.IsDir() {
			folders = append(folders, f.Name())
		}
	}

	return folders, err
}

func TestMultiConfigDiagnose(t *testing.T) {
	tests := []struct {
		description string
		dir         string
		cpDevloop   bool
	}{
		{
			description: "single devloop.yaml outside of source dir",
			dir:         "testdata/diagnose/temp-config",
			cpDevloop:   true,
		},
		{
			description: "multi devloop.yaml outside of source dir",
			dir:         "testdata/diagnose/multi-config",
			cpDevloop:   true,
		},
		{
			description: "multi devloop.yaml",
			dir:         "testdata/diagnose/multi-config",
			cpDevloop:   false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			MarkIntegrationTest(t.T, CanRunWithoutGcp)
			args := []string{}
			if test.cpDevloop {
				tmpDir := t.NewTempDir()
				configContents, err := os.ReadFile(filepath.Join(test.dir, "devloop.yaml"))
				t.CheckNoError(err)
				tmpDir.Write("devloop.yaml", string(configContents))
				args = append(args, fmt.Sprintf("-f=%s", tmpDir.Path("devloop.yaml")))
			}
			out := devloop.Diagnose(append(args, "--yaml-only")...).InDir(test.dir).RunOrFailOutput(t.T)
			templ, err := os.ReadFile(filepath.Join(test.dir, "diagnose.tmpl"))
			t.CheckNoError(err)
			outTemplate := template.Must(template.New("tmpl").Parse(string(templ)))
			cwd, err := filepath.Abs(test.dir)
			t.CheckNoError(err)
			expected := &bytes.Buffer{}
			outTemplate.Execute(expected, map[string]string{"Root": cwd})
			t.CheckDeepEqual(expected.String(), string(out), testutil.YamlObj(t.T))
		})
	}
}
