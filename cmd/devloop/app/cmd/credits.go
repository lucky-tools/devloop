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

package cmd

import (
	"github.com/spf13/cobra"

	"github.com/lucky-tools/devloop/cmd/devloop/app/cmd/credits"
)

func NewCmdCredits() *cobra.Command {
	return NewCmd("credits").
		Hidden(). // internal command
		WithDescription("Export third party notices to given path (./devloop-credits by default)").
		WithExample("export third party licenses to ~/devloop-credits", "credits -d ~/devloop-credits").
		WithFlags([]*Flag{
			{Value: &credits.Path, Name: "dir", Shorthand: "d", DefValue: "./devloop-credits", Usage: "destination directory to place third party licenses"},
		}).
		NoArgs(credits.Export)
}
