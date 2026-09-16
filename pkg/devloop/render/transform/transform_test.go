/*
Copyright 2021 The Skaffold Authors

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

package transform

import (
	"testing"

	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/testutil"
)

func TestNewTransformer(t *testing.T) {
	tests := []struct {
		description string
		config      []latest.Transformer
	}{
		{
			description: "no transform",
			config:      []latest.Transformer{},
		},
		{
			description: "set-labels",
			config: []latest.Transformer{
				{Name: "set-annotations", ConfigMap: []string{"owner:devloop-test"}},
				{Name: "set-labels", ConfigMap: []string{"owner:devloop-test"}},
			},
		},
		{
			description: "values containing ':'",
			config: []latest.Transformer{
				{Name: "apply-setters", ConfigMap: []string{"owner:devloop:test"}},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := NewTransformer(test.config)
			t.CheckNoError(err)
		})
	}
}

func TestNewTransformerDoesNotShareConfigMap(t *testing.T) {
	tests := []struct {
		description string
		config      []latest.Transformer
	}{
		{
			description: "set-labels",
			config: []latest.Transformer{
				{Name: "set-labels", ConfigMap: []string{"owner2:devloop-test"}},
				{Name: "set-labels", ConfigMap: []string{"owner:devloop-test"}},
			},
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			transformer, err := NewTransformer(test.config)
			t.CheckNoError(err)
			t.CheckDeepEqual(1, len(transformer.kptFn[0].ConfigMap))
			t.CheckDeepEqual(1, len(transformer.kptFn[1].ConfigMap))
		})
	}
}

func TestNewValidator_Error(t *testing.T) {
	testutil.Run(t, "", func(t *testutil.T) {
		_, err := NewTransformer([]latest.Transformer{
			{Name: "bad-transformer"},
		})
		t.CheckErrorContains(`unsupported transformer "bad-transformer". please only use the`, err)
	})
}
