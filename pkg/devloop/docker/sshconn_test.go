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

package docker

import (
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/testutil"
)

func TestParseSSHPasswordURL(t *testing.T) {
	key, err := config.GenerateEncryptionKey()
	if err != nil {
		t.Fatal(err)
	}
	enc, err := config.Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		description string
		host        string
		expected    sshPasswordSpec
		expectedOK  bool
		expectedErr bool
	}{
		{
			description: "password with default port",
			host:        "ssh://root:" + enc + "@192.168.1.53",
			expected:    sshPasswordSpec{user: "root", password: "secret", host: "192.168.1.53:22"},
			expectedOK:  true,
		},
		{
			description: "password with explicit port",
			host:        "ssh://root:" + enc + "@192.168.1.53:2222",
			expected:    sshPasswordSpec{user: "root", password: "secret", host: "192.168.1.53:2222"},
			expectedOK:  true,
		},
		{
			description: "password with socket path",
			host:        "ssh://root:" + enc + "@192.168.1.53/var/run/docker.sock",
			expected:    sshPasswordSpec{user: "root", password: "secret", host: "192.168.1.53:22", path: "/var/run/docker.sock"},
			expectedOK:  true,
		},
		{
			description: "ipv6 host",
			host:        "ssh://root:" + enc + "@[::1]:2222",
			expected:    sshPasswordSpec{user: "root", password: "secret", host: "[::1]:2222"},
			expectedOK:  true,
		},
		{
			description: "plain password rejected",
			host:        "ssh://root:secret@192.168.1.53",
			expectedErr: true,
		},
		{
			description: "no password falls back to system ssh",
			host:        "ssh://root@192.168.1.53",
			expectedOK:  false,
		},
		{
			description: "no user",
			host:        "ssh://192.168.1.53",
			expectedOK:  false,
		},
		{
			description: "non-ssh scheme",
			host:        "tcp://root:secret@192.168.1.53",
			expectedOK:  false,
		},
		{
			description: "http host",
			host:        "http://127.0.0.1:8080",
			expectedOK:  false,
		},
		{
			description: "malformed URL",
			host:        "://bad",
			expectedOK:  false,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			spec, ok, err := parseSSHPasswordURL(test.host, key)
			t.CheckError(test.expectedErr, err)
			t.CheckDeepEqual(test.expectedOK, ok)
			if ok {
				t.CheckDeepEqual(test.expected, spec, cmp.AllowUnexported(sshPasswordSpec{}))
			}
		})
	}
}

func TestParseSSHPasswordURLEncrypted(t *testing.T) {
	key, err := config.GenerateEncryptionKey()
	if err != nil {
		t.Fatal(err)
	}
	enc, err := config.Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	testutil.Run(t, "decrypts an enc-prefixed password", func(t *testutil.T) {
		spec, ok, err := parseSSHPasswordURL("ssh://root:"+enc+"@192.168.1.53", key)
		t.CheckError(false, err)
		t.CheckDeepEqual(true, ok)
		t.CheckDeepEqual(sshPasswordSpec{user: "root", password: "secret", host: "192.168.1.53:22"}, spec, cmp.AllowUnexported(sshPasswordSpec{}))
	})

	testutil.Run(t, "fails without a key", func(t *testutil.T) {
		_, ok, err := parseSSHPasswordURL("ssh://root:"+enc+"@192.168.1.53", nil)
		t.CheckError(true, err)
		t.CheckDeepEqual(false, ok)
	})
}

func TestDockerDialStdioCommand(t *testing.T) {
	tests := []struct {
		description string
		path        string
		expected    string
	}{
		{description: "empty path", path: "", expected: "docker system dial-stdio"},
		{description: "root path", path: "/", expected: "docker system dial-stdio"},
		{description: "socket path", path: "/var/run/docker.sock", expected: "docker --host=unix:///var/run/docker.sock system dial-stdio"},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, dockerDialStdioCommand(test.path))
		})
	}
}

func TestBuildctlDialStdioCommand(t *testing.T) {
	tests := []struct {
		description string
		path        string
		expected    string
	}{
		{description: "empty path", path: "", expected: "buildctl dial-stdio"},
		{description: "root path", path: "/", expected: "buildctl dial-stdio"},
		{description: "socket path", path: "/var/run/buildkit/buildkitd.sock", expected: "buildctl --addr=unix:///var/run/buildkit/buildkitd.sock dial-stdio"},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.CheckDeepEqual(test.expected, buildctlDialStdioCommand(test.path))
		})
	}
}

func TestSSHBuildKitDialer(t *testing.T) {
	key, err := config.GenerateEncryptionKey()
	if err != nil {
		t.Fatal(err)
	}
	enc, err := config.Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		description string
		host        string
		expectedOK  bool
		expectedErr bool
	}{
		{description: "password URL", host: "ssh://root:" + enc + "@192.168.1.53", expectedOK: true},
		{description: "password URL with port and path", host: "ssh://root:" + enc + "@192.168.1.53:2222/var/run/buildkit/buildkitd.sock", expectedOK: true},
		{description: "plain password rejected", host: "ssh://root:secret@192.168.1.53", expectedErr: true},
		{description: "no password falls back", host: "ssh://root@192.168.1.53", expectedOK: false},
		{description: "non-ssh scheme", host: "tcp://root:secret@192.168.1.53", expectedOK: false},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			dialer, ok, err := SSHBuildKitDialer(test.host, key)
			t.CheckError(test.expectedErr, err)
			t.CheckDeepEqual(test.expectedOK, ok)
			if ok {
				t.CheckNotNil(dialer)
			} else {
				t.CheckNil(dialer)
			}
		})
	}
}

func TestHostKeyCallback(t *testing.T) {
	tests := []struct {
		description string
		insecure    bool
		withKey     bool
		shouldErr   bool
	}{
		{description: "insecure env skips verification", insecure: true},
		{description: "missing known_hosts errors", shouldErr: true},
		{description: "known_hosts present", withKey: true},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			if test.insecure {
				t.Setenv(sshInsecureHostKeyEnv, "1")
			} else {
				t.Setenv(sshInsecureHostKeyEnv, "")
			}

			home := t.TempDir()
			// os.UserHomeDir consults USERPROFILE on Windows and HOME elsewhere;
			// set both so the test is platform independent.
			t.Setenv("USERPROFILE", home)
			t.Setenv("HOME", home)

			if test.withKey {
				writeKnownHosts(t, home)
			}

			cb, err := hostKeyCallback()
			t.CheckError(test.shouldErr, err)
			if !test.shouldErr {
				t.CheckNotNil(cb)
			}
		})
	}
}

func writeKnownHosts(t *testutil.T, home string) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	line := knownhosts.Line([]string{"127.0.0.1"}, signer.PublicKey())

	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "known_hosts"), []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
}
