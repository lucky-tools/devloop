/*
Copyright 2026 The Skaffold Authors

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
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
)

func NewCmdEncrypt() *cobra.Command {
	return NewCmd("encrypt").
		WithDescription("Encrypt a value with the global Devloop encryption key").
		WithExample("Encrypt an ssh password for use in devloop.yaml", "encrypt secret-password").
		WithFlagAdder(func(f *pflag.FlagSet) {
			f.StringVarP(&opts.GlobalConfig, "config", "c", "", "File for global configurations (defaults to $HOME/.devloop/config)")
		}).
		ExactArgs(1, func(_ context.Context, out io.Writer, args []string) error {
			key, err := config.GetOrCreateEncryptionKey(opts.GlobalConfig)
			if err != nil {
				return err
			}
			encrypted, err := config.Encrypt(key, []byte(args[0]))
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(out, encrypted)
			return err
		})
}
