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
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/lucky-tools/devloop/pkg/devloop/config"
)

const (
	sshDefaultPort = "22"
	sshDialTimeout = 30 * time.Second

	// sshInsecureHostKeyEnv disables ssh host key verification when set to any
	// non-empty value. Only intended for hosts that are not (yet) in known_hosts.
	sshInsecureHostKeyEnv = "DEVLOOP_SSH_INSECURE_HOST_KEY"

	// sshDummyHost is the fake host passed to the docker client. The real
	// connection is established by the ssh dialer, so this value only appears
	// in HTTP requests and is never dialed directly.
	sshDummyHost = "http://docker.example.com"
)

// sshPasswordSpec holds the parsed parts of an ssh:// URL that carries a
// decrypted password. It is only produced when an encrypted password is present;
// hosts without one keep using the Docker CLI connection helper (the system ssh).
type sshPasswordSpec struct {
	user     string
	password string
	host     string // host:port, ready for net.Dial
	path     string
}

// parseSSHPasswordURL parses host. It reports ok=true (with the spec) only when
// host is an ssh:// URL that embeds a password. Otherwise ok=false and the
// caller should fall back to the Docker CLI connection helper.
//
// A password must carry the "enc:" prefix and is decrypted with the global
// encryption key before it is stored in the spec; plain passwords are rejected.
func parseSSHPasswordURL(host string, key []byte) (sshPasswordSpec, bool, error) {
	u, err := url.Parse(host)
	if err != nil || u.Scheme != "ssh" || u.User == nil {
		return sshPasswordSpec{}, false, nil
	}
	password, ok := u.User.Password()
	if !ok {
		return sshPasswordSpec{}, false, nil
	}
	password, err = config.Decrypt(key, password)
	if err != nil {
		return sshPasswordSpec{}, false, fmt.Errorf("decrypting ssh password in %q: %w", host, err)
	}
	hostname := u.Hostname()
	if hostname == "" {
		return sshPasswordSpec{}, false, nil
	}
	port := u.Port()
	if port == "" {
		port = sshDefaultPort
	}
	return sshPasswordSpec{
		user:     u.User.Username(),
		password: password,
		host:     net.JoinHostPort(hostname, port),
		path:     u.Path,
	}, true, nil
}

// sshDialer returns a 2-arg context dialer (the shape used by grpc's
// WithContextDialer) that connects over ssh per spec and bridges stdio via
// remoteCommand.
func sshDialer(spec sshPasswordSpec, remoteCommand string) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		config, err := sshClientConfig(spec)
		if err != nil {
			return nil, err
		}
		client, err := dialSSH(ctx, spec.host, config)
		if err != nil {
			return nil, err
		}
		conn, err := newSSHConn(client, remoteCommand)
		if err != nil {
			_ = client.Close()
			return nil, err
		}
		return conn, nil
	}
}

// nativeSSHDialer returns a dialer that connects to a remote Docker daemon over
// ssh using the password in spec, bridging the Docker HTTP API via
// `docker system dial-stdio`. It adapts sshDialer to the docker client's 3-arg
// dialer shape.
func nativeSSHDialer(spec sshPasswordSpec) func(context.Context, string, string) (net.Conn, error) {
	d := sshDialer(spec, dockerDialStdioCommand(spec.path))
	return func(ctx context.Context, _, _ string) (net.Conn, error) {
		return d(ctx, "")
	}
}

// SSHBuildKitDialer returns a context dialer (for client.WithContextDialer) that
// connects to a remote buildkitd over ssh using the encrypted password in an
// ssh:// URL. ok is false when host is not such a URL, in which case the caller
// should use its default connection logic.
func SSHBuildKitDialer(host string, key []byte) (func(context.Context, string) (net.Conn, error), bool, error) {
	spec, ok, err := parseSSHPasswordURL(host, key)
	if err != nil || !ok {
		return nil, false, err
	}
	return sshDialer(spec, buildctlDialStdioCommand(spec.path)), true, nil
}

func sshClientConfig(spec sshPasswordSpec) (*ssh.ClientConfig, error) {
	hostKeyCallback, err := hostKeyCallback()
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User:            spec.user,
		Auth:            []ssh.AuthMethod{ssh.Password(spec.password)},
		HostKeyCallback: hostKeyCallback,
		Timeout:         sshDialTimeout,
	}, nil
}

// hostKeyCallback returns the standard known_hosts verifier, or an insecure
// no-op verifier when sshInsecureHostKeyEnv is explicitly set.
func hostKeyCallback() (ssh.HostKeyCallback, error) {
	if os.Getenv(sshInsecureHostKeyEnv) != "" {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("locating home directory for ssh known_hosts: %w", err)
	}
	knownHosts := filepath.Join(home, ".ssh", "known_hosts")
	if _, err := os.Stat(knownHosts); err != nil {
		return nil, fmt.Errorf("reading ssh known_hosts %q (run `ssh <host>` once to trust the host key, or set %s=1 to skip verification): %w", knownHosts, sshInsecureHostKeyEnv, err)
	}
	return knownhosts.New(knownHosts)
}

// dialSSH establishes an ssh client connection to addr, honoring ctx during the
// TCP dial and the handshake.
func dialSSH(ctx context.Context, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	conn, err := (&net.Dialer{Timeout: sshDialTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dialing ssh host %q: %w", addr, err)
	}
	// ssh.NewClientConn blocks on the handshake and does not observe ctx. Close
	// the raw connection when ctx is cancelled so the handshake aborts.
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		return nil, fmt.Errorf("ssh handshake with %q: %w", addr, err)
	}
	return ssh.NewClient(sshConn, chans, reqs), nil
}

// dockerDialStdioCommand returns the remote command that bridges the Docker
// HTTP API over the ssh session, mirroring the Docker CLI connection helper.
func dockerDialStdioCommand(path string) string {
	if strings.Trim(path, "/") != "" {
		return "docker --host=unix://" + path + " system dial-stdio"
	}
	return "docker system dial-stdio"
}

// buildctlDialStdioCommand returns the remote command that bridges buildkitd's
// gRPC API over the ssh session, mirroring buildkit's own ssh connection helper.
func buildctlDialStdioCommand(path string) string {
	if strings.Trim(path, "/") != "" {
		return "buildctl --addr=unix://" + path + " dial-stdio"
	}
	return "buildctl dial-stdio"
}

// sshConn adapts an ssh session running `docker system dial-stdio` to a
// net.Conn so it can back an http.Transport. It mirrors the half-close
// semantics of the Docker CLI's commandconn so connection reuse works.
type sshConn struct {
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader

	stderr syncBuffer

	waitErrMu sync.Mutex
	waitErr   error
	waitDone  chan struct{}

	readEOF   atomic.Bool
	writeEOF  atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

func newSSHConn(client *ssh.Client, remoteCommand string) (*sshConn, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	stdin, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	c := &sshConn{
		client:   client,
		session:  session,
		stdin:    stdin,
		stdout:   stdout,
		waitDone: make(chan struct{}),
	}
	session.Stderr = &c.stderr
	if err := session.Start(remoteCommand); err != nil {
		_ = session.Close()
		return nil, err
	}
	go c.wait()
	return c, nil
}

func (c *sshConn) wait() {
	err := c.session.Wait()
	c.waitErrMu.Lock()
	c.waitErr = err
	c.waitErrMu.Unlock()
	close(c.waitDone)
}

func (c *sshConn) Read(p []byte) (int, error) {
	n, err := c.stdout.Read(p)
	if err == io.EOF {
		c.readEOF.Store(true)
		err = c.handleEOF()
		if c.writeEOF.Load() {
			_ = c.Close()
		}
	}
	return n, err
}

// handleEOF waits for the remote command to finish and surfaces a non-nil exit
// status, mirroring commandconn so a failed dial-stdio is not reported as a
// clean connection close.
func (c *sshConn) handleEOF() error {
	<-c.waitDone
	c.waitErrMu.Lock()
	werr := c.waitErr
	c.waitErrMu.Unlock()
	if werr == nil {
		return io.EOF
	}
	return fmt.Errorf("ssh dial-stdio exited: %w: stderr=%s", werr, c.stderr.String())
}

func (c *sshConn) Write(p []byte) (int, error) {
	return c.stdin.Write(p)
}

func (c *sshConn) Close() error {
	c.closeOnce.Do(func() {
		_ = c.session.Close()
		c.closeErr = c.client.Close()
	})
	return c.closeErr
}

func (c *sshConn) CloseRead() error {
	c.readEOF.Store(true)
	if c.writeEOF.Load() {
		return c.Close()
	}
	return nil
}

func (c *sshConn) CloseWrite() error {
	c.writeEOF.Store(true)
	if err := c.stdin.Close(); err != nil {
		return err
	}
	if c.readEOF.Load() {
		return c.Close()
	}
	return nil
}

func (c *sshConn) LocalAddr() net.Addr  { return sshAddr{} }
func (c *sshConn) RemoteAddr() net.Addr { return sshAddr{} }

func (c *sshConn) SetDeadline(time.Time) error      { return nil }
func (c *sshConn) SetReadDeadline(time.Time) error  { return nil }
func (c *sshConn) SetWriteDeadline(time.Time) error { return nil }

type sshAddr struct{}

func (sshAddr) Network() string { return "ssh" }
func (sshAddr) String() string  { return "docker-system-dial-stdio" }

// syncBuffer is a concurrency-safe bytes.Buffer for capturing ssh session
// stderr, which the ssh library writes from its own goroutine.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}
