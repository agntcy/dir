// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agntcy_test

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	runtimeImage   = "golang:1.27.2-bookworm"
	requestTimeout = 15 * time.Second
	commandTimeout = 2 * time.Minute
	startupTimeout = 90 * time.Second
)

type liveFixture struct {
	config           suiteConfig
	prefix, work     string
	ports            map[string]int
	containers       []string
	networkCreated   bool
	client           *http.Client
	proxy            *http.Server
	proxyDone        chan error
	agentKey         *rsa.PrivateKey
	agentID, subject string
	proof            map[string]string
}

func newLiveFixture(cfg suiteConfig) *liveFixture {
	ginkgo.GinkgoHelper()

	work, err := filepath.EvalSymlinks(ginkgo.GinkgoT().TempDir())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	ports := make(map[string]int)
	for offset, name := range []string{"keycloak", "node", "proxy", "verifier", "grpc", "oci"} {
		ports[name] = cfg.portBase + offset
	}

	return &liveFixture{config: cfg, prefix: "dir-agntcy-e2e-" + uuid.NewString(), work: work, ports: ports, client: &http.Client{Timeout: requestTimeout}}
}

func (f *liveFixture) start(ctx context.Context) {
	ginkgo.GinkgoHelper()

	for _, port := range f.ports {
		listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", fmt.Sprintf("0.0.0.0:%d", port))
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "fixture port %d must be free", port)
		gomega.Expect(listener.Close()).To(gomega.Succeed())
	}

	f.certificate()
	key := f.privateKey("result.key")
	f.publicKey("result.pub", &key.PublicKey)
	f.write("verifier.token", []byte("disposable-verifier-token"))
	f.docker(ctx, "network", "create", f.prefix)
	f.networkCreated = true
	f.write("realm.json", []byte(realmJSON))
	// Keycloak runs as a non-root user and must read its disposable realm fixture.
	gomega.Expect(os.Chmod(filepath.Join(f.work, "realm.json"), 0o644)).To(gomega.Succeed())
	f.startContainer(ctx, "postgres", "postgres:16", []string{"-e", "POSTGRES_DB=identity", "-e", "POSTGRES_USER=identity", "-e", "POSTGRES_PASSWORD=disposable-db-password"})
	f.startContainer(ctx, "keycloak", "quay.io/keycloak/keycloak:26.7", []string{
		"-p", f.binding("keycloak", 8080), "-v", f.work + "/realm.json:/opt/keycloak/data/import/realm.json:ro",
		"-e", "KC_BOOTSTRAP_ADMIN_USERNAME=test", "-e", "KC_BOOTSTRAP_ADMIN_PASSWORD=disposable-admin-password",
	}, "start-dev", "--import-realm", "--hostname", f.hostURL("keycloak", "http", "host.docker.internal"))
	gomega.Eventually(ctx, func(ctx context.Context) error {
		_, err := dockerCommand(ctx, "exec", f.prefix+"-postgres", "pg_isready", "-U", "identity")

		return err
	}, startupTimeout, time.Second).Should(gomega.Succeed(), "Postgres startup")
	f.startContainer(ctx, "node", runtimeImage, []string{
		"-v", f.config.node + ":/node:ro", "-p", f.binding("node", 4000),
		"-e", "DB_HOST=postgres", "-e", "DB_PORT=5432", "-e", "DB_NAME=identity",
		"-e", "DB_USERNAME=identity", "-e", "DB_PASSWORD=disposable-db-password", "-e", "DB_USE_SSL=false",
	}, "/node")
	gomega.Eventually(ctx, func(ctx context.Context) error {
		return f.request(ctx, http.MethodGet, f.hostURL("keycloak", "http", "127.0.0.1")+"/realms/primary/.well-known/openid-configuration", nil, nil)
	}, startupTimeout, time.Second).Should(gomega.Succeed(), "Keycloak startup")
	f.startProxy(ctx)
	f.enroll(ctx)
	f.startContainer(ctx, "verifier", runtimeImage, []string{
		"-v", f.config.verifier + ":/verifier:ro", "-v", f.work + ":/fixtures:ro", "-p", f.binding("verifier", 8443),
		"-e", "IDENTITY_NODE_URL=" + f.hostURL("proxy", "https", "host.docker.internal"),
		"-e", "SSL_CERT_FILE=/fixtures/tls.crt", "-e", "RESULT_SIGNING_KEY_PATH=/fixtures/result.key",
		"-e", "VERIFIER_BEARER_TOKEN_FILE=/fixtures/verifier.token", "-e", "TLS_CERT_FILE=/fixtures/tls.crt",
		"-e", "TLS_KEY_FILE=/fixtures/tls.key", "-e", "RESULT_TTL=20s",
	}, "/verifier")
	f.write("daemon.yaml", []byte(daemonConfig))
	f.startContainer(ctx, "directory", runtimeImage, []string{
		"-v", f.config.dirctl + ":/work/dirctl:ro", "-v", f.work + ":/fixtures",
		"-p", f.binding("grpc", 8888), "-p", f.binding("oci", 5555), "-e", "SSL_CERT_FILE=/fixtures/tls.crt",
	}, "/work/dirctl", "daemon", "start", "--config", "/fixtures/daemon.yaml", "--data-dir", "/state")
	gomega.Eventually(ctx, func(ctx context.Context) error {
		_, err := f.cli(ctx, "search", "--name", "example.com/agntcy-e2e/*")

		return err
	}, startupTimeout, time.Second).Should(gomega.Succeed(), "Directory daemon startup")
}

func (f *liveFixture) binding(name string, containerPort int) string {
	return fmt.Sprintf("%d:%d", f.ports[name], containerPort)
}

func (f *liveFixture) hostURL(name, scheme, host string) string {
	return scheme + "://" + net.JoinHostPort(host, strconv.Itoa(f.ports[name]))
}

func (f *liveFixture) startContainer(ctx context.Context, name, image string, options []string, args ...string) {
	ginkgo.GinkgoHelper()

	container := f.prefix + "-" + name
	f.containers = append(f.containers, container)
	commandArgs := []string{"run", "-d", "--name", container, "--network", f.prefix, "--network-alias", name, "--add-host", "host.docker.internal:host-gateway"}
	commandArgs = append(commandArgs, options...)
	commandArgs = append(commandArgs, image)
	f.docker(ctx, append(commandArgs, args...)...)
}

func (f *liveFixture) docker(ctx context.Context, args ...string) string {
	ginkgo.GinkgoHelper()

	output, err := dockerCommand(ctx, args...)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return output
}

func dockerCommand(ctx context.Context, args ...string) (string, error) {
	bounded, cancel := context.WithTimeout(ctx, commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(bounded, "docker", args...)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker %v: %w\n%s%s", args, err, output, stderr.String())
	}

	return strings.TrimSpace(string(output)), nil
}

func (f *liveFixture) cli(ctx context.Context, args ...string) (string, error) {
	bounded, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	cliArgs := []string{"exec", f.prefix + "-directory", "/work/dirctl", "--server-addr", "localhost:8888"}

	return dockerCommand(bounded, append(cliArgs, args...)...)
}

func (f *liveFixture) mustCLI(ctx context.Context, args ...string) string {
	ginkgo.GinkgoHelper()

	output, err := f.cli(ctx, args...)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return output
}

func (f *liveFixture) write(name string, data []byte) {
	ginkgo.GinkgoHelper()
	gomega.Expect(os.WriteFile(filepath.Join(f.work, name), data, 0o600)).To(gomega.Succeed())
}

func (f *liveFixture) request(ctx context.Context, method, endpoint string, body, out any) error {
	var input io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}

		input = bytes.NewReader(data)
	}

	data, err := f.fetch(ctx, method, endpoint, "application/json", input)
	if err != nil {
		return err
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("decode %s: %w", endpoint, err)
		}
	}

	return nil
}

func (f *liveFixture) fetch(ctx context.Context, method, endpoint, contentType string, input io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, input)
	if err != nil {
		return nil, fmt.Errorf("create fixture request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json, application/json")

	response, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s: %w", endpoint, err)
	}
	defer response.Body.Close() //nolint:errcheck // ReadAll reports response errors.

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", endpoint, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%s returned %d: %s", endpoint, response.StatusCode, data)
	}

	return data, nil
}

func (f *liveFixture) startProxy(ctx context.Context) {
	ginkgo.GinkgoHelper()

	target, err := url.Parse(f.hostURL("node", "http", "127.0.0.1"))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// TLS termination only: all identity and credential operations reach Node.
	proxy := httputil.NewSingleHostReverseProxy(target)
	transport := &http.Transport{ResponseHeaderTimeout: requestTimeout}
	ginkgo.DeferCleanup(transport.CloseIdleConnections)
	proxy.Transport = transport
	cert, err := tls.LoadX509KeyPair(filepath.Join(f.work, "tls.crt"), filepath.Join(f.work, "tls.key"))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort("0.0.0.0", strconv.Itoa(f.ports["proxy"])))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	f.proxy = &http.Server{Handler: proxy, ReadHeaderTimeout: requestTimeout, ReadTimeout: requestTimeout, WriteTimeout: requestTimeout}

	f.proxyDone = make(chan error, 1)
	go func() {
		f.proxyDone <- f.proxy.Serve(tls.NewListener(listener, &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{cert}}))
	}()
}

func (f *liveFixture) cleanup() {
	ginkgo.GinkgoHelper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var failures []error

	for _, container := range f.containers {
		output, err := exec.CommandContext(ctx, "docker", "logs", container).CombinedOutput()

		logs := string(output)
		if err != nil {
			logs += "\n" + err.Error()
		}

		name := strings.TrimPrefix(container, f.prefix+"-") + ".log"
		if err := os.WriteFile(filepath.Join(f.config.artifacts, name), []byte(logs), 0o600); err != nil {
			failures = append(failures, err)
		}
	}

	if f.proxy != nil {
		if err := f.proxy.Shutdown(ctx); err != nil {
			failures = append(failures, err)
		}

		if err := <-f.proxyDone; err != nil && !errors.Is(err, http.ErrServerClosed) {
			failures = append(failures, err)
		}
	}

	for _, v := range slices.Backward(f.containers) {
		if _, err := dockerCommand(ctx, "rm", "-f", v); err != nil {
			failures = append(failures, err)
		}
	}

	if f.networkCreated {
		if _, err := dockerCommand(ctx, "network", "rm", f.prefix); err != nil {
			failures = append(failures, err)
		}
	}

	f.client.CloseIdleConnections()
	gomega.Expect(errors.Join(failures...)).To(gomega.Succeed(), "fixture cleanup")
}

const realmJSON = `{
 "realm": "primary", "enabled": true,
 "clients": [{"clientId":"primary-client", "secret":"disposable-client-secret", "enabled":true,
 "publicClient":false, "serviceAccountsEnabled":true, "standardFlowEnabled":false, "protocol":"openid-connect",
 "protocolMappers":[{"name":"identity-audience","protocol":"openid-connect","protocolMapper":"oidc-audience-mapper",
 "config":{"included.custom.audience":"identity-node","access.token.claim":"true"}}]}]
}`

const daemonConfig = `server:
  listen_address: ':8888'
  store:
    provider: oci
    oci:
      local_dir: /state/store
      registry_address: 0.0.0.0:5555
      repository_name: dir
      auth_config:
        insecure: true
  routing:
    listen_address: /ip4/127.0.0.1/tcp/0
    key_path: /state/node.key
  database:
    type: sqlite
    sqlite:
      path: /state/dir.db
  extractor:
    asset_dir: /nonexistent-extractor-assets-e2e
  publication:
    scheduler_interval: 1h
    worker_count: 1
    worker_timeout: 30m
reconciler:
  local_registry:
    registry_address: localhost:5555
    repository_name: dir
    auth_config:
      insecure: true
  indexer:
    enabled: true
    interval: 2s
  identity:
    enabled: true
    interval: 2s
    record_timeout: 15s
    agntcy:
      verifier_url: https://verifier:8443
      verifier_trust_bundle_file: /fixtures/result.pub
      bearer_token_file: /fixtures/verifier.token
      require_agent_badge: true
      request_timeout: 10s
      max_verification_age: 20s
`
