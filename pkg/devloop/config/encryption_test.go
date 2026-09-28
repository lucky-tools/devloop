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

package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lucky-tools/devloop/testutil"
)

func TestEncryptDecrypt(t *testing.T) {
	testutil.Run(t, "round trip", func(t *testutil.T) {
		key, err := GenerateEncryptionKey()
		t.CheckError(false, err)

		enc, err := Encrypt(key, []byte("ssh secret"))
		t.CheckError(false, err)
		t.CheckDeepEqual(true, strings.HasPrefix(enc, "enc:"))

		dec, err := Decrypt(key, enc)
		t.CheckError(false, err)
		t.CheckDeepEqual("ssh secret", dec)
	})

	testutil.Run(t, "plain value rejected", func(t *testutil.T) {
		_, err := Decrypt(nil, "not-encrypted")
		t.CheckError(true, err)
	})

	testutil.Run(t, "wrong key fails", func(t *testutil.T) {
		key, err := GenerateEncryptionKey()
		t.CheckError(false, err)
		other, err := GenerateEncryptionKey()
		t.CheckError(false, err)
		enc, err := Encrypt(key, []byte("secret"))
		t.CheckError(false, err)

		_, err = Decrypt(other, enc)
		t.CheckError(true, err)
	})

	testutil.Run(t, "no key fails on encrypted value", func(t *testutil.T) {
		key, err := GenerateEncryptionKey()
		t.CheckError(false, err)
		enc, err := Encrypt(key, []byte("secret"))
		t.CheckError(false, err)

		_, err = Decrypt(nil, enc)
		t.CheckError(true, err)
	})
}

func TestGetOrCreateEncryptionKey(t *testing.T) {
	testutil.Run(t, "generates and persists a key", func(t *testutil.T) {
		configFile := filepath.Join(t.TempDir(), "config")

		key, err := GetOrCreateEncryptionKey(configFile)
		t.CheckError(false, err)
		t.CheckDeepEqual(encryptionKeySize, len(key))

		got, err := GetEncryptionKey(configFile)
		t.CheckError(false, err)
		t.CheckDeepEqual(key, got)
	})

	testutil.Run(t, "missing config returns nil key", func(t *testutil.T) {
		got, err := GetEncryptionKey(filepath.Join(t.TempDir(), "missing"))
		t.CheckError(false, err)
		t.CheckDeepEqual(0, len(got))
	})
}
