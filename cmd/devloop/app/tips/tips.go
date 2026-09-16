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

package tips

import (
	"io"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/output"
)

// PrintForRun prints tips to the user who has run `devloop run`.
func PrintForRun(out io.Writer, opts config.DevloopOptions) {
	if !opts.Tail {
		printTip(out, "You can also run [devloop run --tail] to get the logs")
	}
}

// PrintForInit prints tips to the user who has run `devloop init`.
func PrintForInit(out io.Writer, opts config.DevloopOptions) {
	printTip(out, "You can now run [devloop build] to build the artifacts")
	printTip(out, "or [devloop run] to build and deploy")
	printTip(out, "or [devloop dev] to enter development mode, with auto-redeploy")
}

// PrintForTest prints tips on when to use devloop test.
func PrintForTest(out io.Writer) {
	printTip(out, "You need to:")
	printTip(out, "run [devloop test] with [--build-artifacts <file-output>] for running tests on artifacts from a given file.")
}

// PrintUseRunVsDeploy prints tips on when to use devloop run vs deploy.
func PrintUseRunVsDeploy(out io.Writer) {
	printTip(out, "You either need to:")
	printTip(out, "run [devloop deploy] with [--images TAG] for each pre-built artifact")
	printTip(out, "or [devloop run] instead, to let Devloop build, tag and deploy artifacts.")
}

// PrintUseBuildAndExec prints tip to use artifacts from previous build in devloop exec.
func PrintUseBuildAndExec(out io.Writer) {
	printTip(out, "Check all the images have a tag assigned:")
	printTip(out, "run [devloop exec] with [--build-artifacts <file-output>] for running an action using images from a previous build")
}

func printTip(out io.Writer, message string) {
	output.Green.Fprintln(out, message)
}
