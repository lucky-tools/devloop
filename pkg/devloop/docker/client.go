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
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/docker/cli/cli/connhelper"
	"github.com/docker/go-connections/tlsconfig"
	dockerspec "github.com/moby/docker-image-spec/specs-go/v1"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/lucky-tools/devloop/pkg/devloop/cluster"
	"github.com/lucky-tools/devloop/pkg/devloop/config"
	"github.com/lucky-tools/devloop/pkg/devloop/output/log"
	"github.com/lucky-tools/devloop/pkg/devloop/util"
	"github.com/lucky-tools/devloop/pkg/devloop/version"
)

// minikube 1.13.0 renumbered exit codes
const (
	minikubeDriverConfictExitCode = 51
	minikubeExGuestUnavailable    = 89
	oldMinikubeBadUsageExitCode   = 64
)

// For testing
var (
	NewAPIClient = NewAPIClientImpl
)

var (
	dockerAPIClientOnce sync.Once
	dockerAPIClient     LocalDaemon
	dockerAPIClientErr  error
)

type Config interface {
	Prune() bool
	ContainerDebugging() bool
	GlobalConfig() string
	GetKubeContext() string
	MinikubeProfile() string
	GetInsecureRegistries() map[string]bool
	Mode() config.RunMode
}

// NewAPIClientImpl guesses the docker client to use based on current Kubernetes context.
func NewAPIClientImpl(ctx context.Context, cfg Config) (LocalDaemon, error) {
	dockerAPIClientOnce.Do(func() {
		key, err := config.GetOrCreateEncryptionKey(cfg.GlobalConfig())
		if err != nil {
			dockerAPIClientErr = err
			return
		}
		env, apiClient, err := newAPIClient(ctx, cfg.GetKubeContext(), cfg.MinikubeProfile(), key)
		dockerAPIClient = NewLocalDaemon(apiClient, env, cfg.Prune(), cfg)
		dockerAPIClientErr = err
	})

	return dockerAPIClient, dockerAPIClientErr
}

// TODO(https://github.com/lucky-tools/devloop/issues/3668):
// remove minikubeProfile from here and instead detect it by matching the
// kubecontext API Server to minikube profiles

// newAPIClient guesses the docker client to use based on current Kubernetes context.
func newAPIClient(ctx context.Context, kubeContext string, minikubeProfile string, key []byte) ([]string, client.APIClient, error) {
	if minikubeProfile != "" { // skip validation if explicitly specifying minikubeProfile.
		return newMinikubeAPIClient(ctx, minikubeProfile, key)
	}
	if cluster.GetClient().IsMinikube(ctx, kubeContext) {
		return newMinikubeAPIClient(ctx, kubeContext, key)
	}
	return newEnvAPIClient(key)
}

// connectionHelperOpts returns client options that connect to the given host
// through a Docker connection helper when one is registered for the host's
// scheme (currently only ssh://). The bool result reports whether a helper was
// found; when false, the caller should connect to host directly.
func connectionHelperOpts(host string, key []byte) ([]client.Opt, bool, error) {
	// ssh:// URLs that embed a password are dialed natively, since the Docker
	// CLI connection helper rejects plain-text passwords and disables the tty
	// needed for an interactive password prompt. The password must be encrypted
	// with the global encryption key (see `devloop encrypt`).
	spec, ok, err := parseSSHPasswordURL(host, key)
	if err != nil {
		return nil, false, err
	}
	if ok {
		// WithHost must come before WithDialContext: WithHost calls
		// sockets.ConfigureTransport, which for an "http://" scheme replaces the
		// transport's DialContext with the default net.Dialer. Applying the ssh
		// dialer last ensures it survives and the dummy host is never dialed.
		return []client.Opt{
			client.WithHost(sshDummyHost),
			client.WithDialContext(nativeSSHDialer(spec)),
		}, true, nil
	}
	helper, err := connhelper.GetConnectionHelper(host)
	if err != nil || helper == nil {
		return nil, false, nil
	}
	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: helper.Dialer,
		},
	}
	return []client.Opt{
		client.WithHTTPClient(httpClient),
		client.WithHost(helper.Host),
		client.WithDialContext(helper.Dialer),
	}, true, nil
}

// newEnvAPIClient returns a docker client based on the environment variables set.
// It will "negotiate" the highest possible API version supported by both the client
// and the server if there is a mismatch.
func newEnvAPIClient(key []byte) ([]string, client.APIClient, error) {
	opts := []client.Opt{client.WithHTTPHeaders(getUserAgentHeader())}
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		helperOpts, ok, err := connectionHelperOpts(host, key)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			opts = append(opts, helperOpts...)
		} else {
			opts = append(opts, client.FromEnv)
		}
	} else {
		log.Entry(context.TODO()).Infof("DOCKER_HOST env is not set, using the host from docker context.")

		command := exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}")
		out, err := util.RunCmdOut(context.TODO(), command)
		if err != nil {
			// docker cli not installed.
			log.Entry(context.TODO()).Warnf("Could not get docker context: %s, falling back to the default docker host", err)
		} else {
			s := strings.TrimSpace(string(out))
			// output can be empty if user uses docker as alias for podman
			if len(s) > 0 {
				helperOpts, ok, err := connectionHelperOpts(s, key)
				if err != nil {
					return nil, nil, err
				}
				if ok {
					opts = append(opts, helperOpts...)
				} else {
					opts = append(opts, client.WithHost(s))
				}
			}
		}
	}

	opts = append(opts, client.WithAPIVersionNegotiation())
	cli, err := client.New(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("error getting docker client: %s", err)
	}

	return nil, cli, nil
}

// ResolveDockerHost returns the Docker daemon host the process is configured to
// use: the DOCKER_HOST environment variable if set, otherwise the host from the
// active docker context. It does not apply minikube detection. An empty result
// means the caller should fall back to its platform default.
func ResolveDockerHost(ctx context.Context) string {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return host
	}
	out, err := util.RunCmdOut(ctx, exec.Command("docker", "context", "inspect", "--format", "{{.Endpoints.docker.Host}}"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

type ExitCoder interface {
	ExitCode() int
}

// newMinikubeAPIClient returns a docker client using the environment variables
// provided by minikube.
func newMinikubeAPIClient(ctx context.Context, minikubeProfile string, key []byte) ([]string, client.APIClient, error) {
	env, err := getMinikubeDockerEnv(ctx, minikubeProfile)
	if err != nil {
		// When minikube uses the infamous `none` driver, `minikube docker-env` will exit with
		// code 51 (>= 1.13.0) or 64 (< 1.13.0).  Note that exit code 51 was unused prior to 1.13.0
		// so it is safe to check here without knowing the minikube version.
		var exitError ExitCoder
		if errors.As(err, &exitError) && (exitError.ExitCode() == minikubeDriverConfictExitCode || exitError.ExitCode() == oldMinikubeBadUsageExitCode || exitError.ExitCode() == minikubeExGuestUnavailable) {
			// Let's ignore the error and fall back to local docker daemon.
			log.Entry(context.TODO()).Warnf("Could not get minikube docker env, falling back to local docker daemon: %s", err)
			return newEnvAPIClient(key)
		}

		return nil, nil, err
	}

	var httpclient *http.Client
	if dockerCertPath := env["DOCKER_CERT_PATH"]; dockerCertPath != "" {
		options := tlsconfig.Options{
			CAFile:             filepath.Join(dockerCertPath, "ca.pem"),
			CertFile:           filepath.Join(dockerCertPath, "cert.pem"),
			KeyFile:            filepath.Join(dockerCertPath, "key.pem"),
			InsecureSkipVerify: env["DOCKER_TLS_VERIFY"] == "",
		}
		tlsc, err := tlsconfig.Client(options)
		if err != nil {
			return nil, nil, err
		}

		httpclient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsc,
			},
			CheckRedirect: client.CheckRedirect,
		}
	}

	host := env["DOCKER_HOST"]
	if host == "" {
		host = client.DefaultDockerHost
	}

	api, err := client.New(
		client.WithHost(host),
		client.WithHTTPClient(httpclient),
		client.WithHTTPHeaders(getUserAgentHeader()),
		client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, nil, err
	}

	if host != client.DefaultDockerHost {
		log.Entry(context.TODO()).Infof("Using minikube docker daemon at %s", host)
	}

	// Keep the minikube environment variables
	var environment []string
	for k, v := range env {
		environment = append(environment, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(environment)

	return environment, api, err
}

func getUserAgentHeader() map[string]string {
	userAgent := fmt.Sprintf("devloop-%s", version.Get().Version)
	log.Entry(context.TODO()).Debugf("setting Docker user agent to %s", userAgent)
	return map[string]string{
		"User-Agent": userAgent,
	}
}

func getMinikubeDockerEnv(ctx context.Context, minikubeProfile string) (map[string]string, error) {
	if minikubeProfile == "" {
		return nil, fmt.Errorf("empty minikube profile")
	}
	cmd, err := cluster.GetClient().MinikubeExec(ctx, "docker-env", "--shell", "none", "-p", minikubeProfile)
	if err != nil {
		return nil, fmt.Errorf("executing minikube command: %w", err)
	}
	out, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("getting minikube env: %w", err)
	}

	env := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("unable to parse minikube docker-env keyvalue: %s, line: %s, output: %s", kv, line, string(out))
		}
		if kv[1] == "" {
			continue
		}
		env[kv[0]] = kv[1]
	}

	return env, nil
}

// This was copied from api/server/router/image/image_routes.go, since it's not
// exported. The ImageInspect API now returns a dockerspec.DockerOCIImageConfig,
// whereas before it used to return a container.Config, so we need to convert it
// before using it to call ContainerCreate.
func OCIImageConfigToContainerConfig(img string, cfg *dockerspec.DockerOCIImageConfig) *container.Config {
	exposedPorts := make(network.PortSet, len(cfg.ExposedPorts))
	for k, v := range cfg.ExposedPorts {
		p, err := network.ParsePort(k)
		if err == nil {
			exposedPorts[p] = v
		}
	}

	return &container.Config{
		Image:        img,
		Entrypoint:   cfg.Entrypoint,
		Env:          cfg.Env,
		Cmd:          cfg.Cmd,
		User:         cfg.User,
		WorkingDir:   cfg.WorkingDir,
		ExposedPorts: exposedPorts,
		Volumes:      cfg.Volumes,
		Labels:       cfg.Labels,
		ArgsEscaped:  cfg.ArgsEscaped, //nolint:staticcheck // Ignore SA1019. Need to keep it in image.
		StopSignal:   cfg.StopSignal,
		Healthcheck:  cfg.Healthcheck,
		OnBuild:      cfg.OnBuild,
		Shell:        cfg.Shell,
	}
}
