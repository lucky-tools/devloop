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

package instrumentation

import (
	"context"

	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
)

// errHandler prints errors to logrus.
type errHandler struct{}

func (h errHandler) Handle(err error) {
	log.Entry(context.TODO()).Debugf("Error with tracing: %v", err)
}
