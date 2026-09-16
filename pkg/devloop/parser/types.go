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

package parser

import (
	"github.com/lucky-tools/devloop/pkg/devloop/parser/configlocations"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
)

// DevloopConfigSet encapsulates a slice of devloop configurations.
type DevloopConfigSet []*DevloopConfigEntry

// DevloopConfigEntry encapsulates a single devloop configuration, along with the source filename and its index in that file.
type DevloopConfigEntry struct {
	*latest.DevloopConfig
	SourceFile   string
	SourceIndex  int
	IsRootConfig bool
	IsRemote     bool
	YAMLInfos    *configlocations.YAMLInfos
}

// SelectRootConfigs filters DevloopConfigSet to only configs read from the root devloop.yaml file
func (s DevloopConfigSet) SelectRootConfigs() DevloopConfigSet {
	var filteredSet DevloopConfigSet
	for _, entry := range s {
		if entry.IsRootConfig {
			filteredSet = append(filteredSet, entry)
		}
	}
	return filteredSet
}

// Locate gets the location for a devloop schema struct pointer
func (s DevloopConfigSet) Locate(obj interface{}) *configlocations.Location {
	loc := configlocations.MissingLocation()
	for _, c := range s {
		if l := c.YAMLInfos.Locate(obj); l.StartLine != -1 {
			loc = l
		}
	}
	return loc
}

// LocateField gets the location for a devloop field from a devloop schema struct pointer and a field name
func (s DevloopConfigSet) LocateField(obj interface{}, fieldName string) *configlocations.Location {
	loc := configlocations.MissingLocation()
	for _, c := range s {
		if l := c.YAMLInfos.LocateField(obj, fieldName); l.StartLine != -1 {
			loc = l
		}
	}
	return loc
}

// LocateElement gets the location for a devloop element from a devloop schema struct pointer and a slice/array index(int)
func (s DevloopConfigSet) LocateElement(obj interface{}, idx int) *configlocations.Location {
	loc := configlocations.MissingLocation()
	for _, c := range s {
		if l := c.YAMLInfos.LocateElement(obj, idx); l.StartLine != -1 {
			loc = l
		}
	}
	return loc
}
