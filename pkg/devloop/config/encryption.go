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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"

	"github.com/lucky-tools/devloop/pkg/devloop/yaml"
)

const (
	// encryptionPrefix marks a value as encrypted with the global encryption key.
	encryptionPrefix = "enc:"

	// encryptionKeySize is the AES-256 key length in bytes.
	encryptionKeySize = 32
)

// GenerateEncryptionKey returns a fresh random 256-bit key for use with
// Encrypt/Decrypt.
func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, encryptionKeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("generating encryption key: %w", err)
	}
	return key, nil
}

// Encrypt encrypts plaintext under key with AES-256-GCM, returning an
// "enc:"-prefixed URL-safe base64 string that can be embedded in a devloop.yaml
// value (e.g. the password part of an `ssh://user:...@host` URL).
func Encrypt(key []byte, plaintext []byte) (string, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return encryptionPrefix + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// Decrypt reverses Encrypt. Values without the "enc:" prefix are rejected, so a
// value that must stay secret (e.g. an ssh password) is required to be
// ciphertext. An encrypted value cannot be decrypted without a configured key,
// which is reported as an error.
func Decrypt(key []byte, value string) (string, error) {
	if !strings.HasPrefix(value, encryptionPrefix) {
		return "", errors.New("value must be encrypted: run `devloop encrypt <value>` and use the resulting enc:... value")
	}
	if len(key) == 0 {
		return "", errors.New("value is encrypted but no encryption key is configured in the global Devloop config (run `devloop encrypt <value>` to create one)")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, encryptionPrefix))
	if err != nil {
		return "", fmt.Errorf("decoding encrypted value: %w", err)
	}
	aead, err := newAEAD(key)
	if err != nil {
		return "", err
	}
	if len(raw) < aead.NonceSize() {
		return "", errors.New("encrypted value is too short")
	}
	nonce, ciphertext := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting value (wrong key?): %w", err)
	}
	return string(plaintext), nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating AES cipher (key must be %d bytes): %w", encryptionKeySize, err)
	}
	return cipher.NewGCM(block)
}

// GetEncryptionKey returns the configured encryption key, or (nil, nil) when
// none is usable. It never creates or writes the key and is lenient about an
// unreadable global config; only a present-but-invalid key is reported as an
// error.
func GetEncryptionKey(configFile string) ([]byte, error) {
	path, err := resolveGlobalConfigPath(configFile)
	if err != nil {
		return nil, nil
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	cfg := GlobalConfig{}
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return nil, nil
	}
	return decodeEncryptionKey(cfg.EncryptionKey)
}

// GetOrCreateEncryptionKey returns the configured key, generating and persisting
// a fresh one in the global config when none is set.
func GetOrCreateEncryptionKey(configFile string) ([]byte, error) {
	key, err := GetEncryptionKey(configFile)
	if err != nil {
		return nil, err
	}
	if key != nil {
		return key, nil
	}
	key, err = GenerateEncryptionKey()
	if err != nil {
		return nil, err
	}

	path, err := resolveGlobalConfigPath(configFile)
	if err != nil {
		return nil, err
	}
	// Preserve any existing settings when adding the key.
	cfg := GlobalConfig{}
	if contents, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(contents, &cfg); err != nil {
			return nil, fmt.Errorf("unmarshalling global config: %w", err)
		}
	}
	cfg.EncryptionKey = base64.StdEncoding.EncodeToString(key)

	contents, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o744); err != nil {
		return nil, fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		return nil, fmt.Errorf("writing config file: %w", err)
	}
	return key, nil
}

func decodeEncryptionKey(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, nil
	}
	key, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decoding encryption key from global config: %w", err)
	}
	if len(key) != encryptionKeySize {
		return nil, fmt.Errorf("global config encryption key must be %d bytes (base64), got %d", encryptionKeySize, len(key))
	}
	return key, nil
}

// resolveGlobalConfigPath returns the global config path without creating it.
func resolveGlobalConfigPath(configFile string) (string, error) {
	if configFile != "" {
		return configFile, nil
	}
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("retrieving home directory: %w", err)
	}
	return filepath.Join(home, defaultConfigDir, defaultConfigFile), nil
}
