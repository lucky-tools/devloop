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
	"errors"
	"fmt"
	"io"
	"strings"

	"gopkg.in/yaml.v3"

	sErrors "github.com/lucky-tools/devloop/pkg/devloop/schema/errors"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/latest"
	"github.com/lucky-tools/devloop/pkg/devloop/schema/util"
	misc "github.com/lucky-tools/devloop/pkg/devloop/util"
)

// AllVersions refers to all the supported API schema versions.
var AllVersions = Versions{
	{latest.Version, latest.NewDevloopConfig},
}

type APIVersion struct {
	Version string `yaml:"apiVersion"`
}

type Version struct {
	APIVersion string
	Factory    func() util.VersionedConfig
}

type Versions []Version

// Find search the constructor for a given api version.
func (v *Versions) Find(apiVersion string) (func() util.VersionedConfig, bool) {
	for _, version := range *v {
		if version.APIVersion == apiVersion {
			return version.Factory, true
		}
	}

	return nil, false
}

// IsDevloopConfig is for determining if a file is devloop config file.
func IsDevloopConfig(file string) bool {
	if config, err := ParseConfig(file); err == nil && config != nil {
		return true
	}
	return false
}

// ParseConfig reads a configuration file.
func ParseConfig(filename string) ([]util.VersionedConfig, error) {
	buf, err := misc.ReadConfiguration(filename)
	if err != nil {
		return nil, fmt.Errorf("read devloop config: %w", err)
	}
	factories, err := configFactoryFromAPIVersion(buf)
	if err != nil {
		return nil, err
	}
	buf, err = removeYamlAnchors(buf)
	if err != nil {
		return nil, fmt.Errorf("unable to re-marshal YAML without dotted keys: %w", err)
	}
	return parseConfig(buf, factories)
}

// ParseConfigAndUpgrade reads a configuration file. Devloop supports a single
// schema version, so no upgrade is performed.
func ParseConfigAndUpgrade(filename string) ([]util.VersionedConfig, error) {
	return ParseConfig(filename)
}

// configFactoryFromAPIVersion checks that all configs in the input stream have the same API version, and returns a function to create a config with that API version.
func configFactoryFromAPIVersion(buf []byte) ([]func() util.VersionedConfig, error) {
	// This is to quickly check that it's possibly a devloop.yaml,
	// without parsing the whole file.
	if !bytes.Contains(buf, []byte("apiVersion")) {
		return nil, errors.New("missing apiVersion")
	}

	var factories []func() util.VersionedConfig
	b := bytes.NewReader(buf)
	decoder := yaml.NewDecoder(b)
	for {
		var v APIVersion
		err := decoder.Decode(&v)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing api version: %w", err)
		}
		factory, present := AllVersions.Find(v.Version)
		if !present {
			return nil, sErrors.ConfigUnknownAPIVersionErr(v.Version)
		}
		factories = append(factories, factory)
	}
	return factories, nil
}

// removeYamlAnchors removes all top-level keys starting with `.` from the input stream so they can be used as YAML anchors
func removeYamlAnchors(buf []byte) ([]byte, error) {
	in := bytes.NewReader(buf)
	var out bytes.Buffer

	decoder := yaml.NewDecoder(in)
	decoder.KnownFields(true)
	encoder := yaml.NewEncoder(&out)
	for {
		parsed := make(map[string]interface{})
		err := decoder.Decode(parsed)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to parse YAML: %w", err)
		}
		for field := range parsed {
			if strings.HasPrefix(field, ".") {
				delete(parsed, field)
			}
		}
		err = encoder.Encode(parsed)
		if err != nil {
			return nil, err
		}
	}
	err := encoder.Close()
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func parseConfig(buf []byte, factories []func() util.VersionedConfig) ([]util.VersionedConfig, error) {
	b := bytes.NewReader(buf)
	decoder := yaml.NewDecoder(b)
	decoder.KnownFields(true)
	var cfgs []util.VersionedConfig
	for index := 0; index < len(factories); index++ {
		cfg := factories[index]()
		err := decoder.Decode(cfg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to parse config: %w", err)
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}

// UpgradeTo returns the configs unchanged when they are already at toVersion.
// Devloop supports a single schema version, so configs at any other version
// cannot be upgraded.
func UpgradeTo(configs []util.VersionedConfig, toVersion string) ([]util.VersionedConfig, error) {
	for _, cfg := range configs {
		if cfg.GetVersion() != toVersion {
			return nil, fmt.Errorf("config version %q cannot be upgraded to %q: only %q is supported", cfg.GetVersion(), toVersion, latest.Version)
		}
	}
	return configs, nil
}
