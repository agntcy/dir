// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliconfig "github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSummarizeAndHasFailure(t *testing.T) {
	results := []checkResult{
		{Name: "pass", Status: statusPass},
		{Name: "fail", Status: statusFail},
		{Name: "warn", Status: statusWarn},
		{Name: "skip", Status: statusSkip},
	}

	summary := summarize(results)

	assert.Equal(t, checkSummary{
		Total:   4,
		Passed:  1,
		Failed:  1,
		Warned:  1,
		Skipped: 1,
	}, summary)
	assert.True(t, hasFailure(results))
	assert.False(t, hasFailure([]checkResult{{Name: "warn", Status: statusWarn}}))
}

func TestRunDoctorReportsConfigFailureAsResult(t *testing.T) {
	resetDoctorTestState(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	output := runDoctor(context.Background(), newDoctorTestCommand())

	require.NotEmpty(t, output.Results)
	assert.Equal(t, statusFail, output.Results[0].Status)
	assert.Equal(t, "context_config", output.Results[0].Name)
	assert.Equal(t, 1, output.Summary.Failed)
	assert.True(t, hasFailure(output.Results))
}

func TestCanRunCoreChecks(t *testing.T) {
	assert.True(t, canRunCoreChecks(checkResult{Status: statusPass}))
	assert.True(t, canRunCoreChecks(checkResult{Status: statusWarn}))
	assert.False(t, canRunCoreChecks(checkResult{Status: statusFail}))
}

func TestRunDoctorUsesCheckPipelineWhenConfigPasses(t *testing.T) {
	resetDoctorTestState(t)

	runner := newDoctorRunner(doctorDeps{
		directoryAPI: func(context.Context, string, time.Duration) checkResult {
			return checkResult{Name: "directory_api_tcp", Status: statusPass}
		},
		directoryClient: func(context.Context, *client.Config) (closer, checkResult) {
			return noopDoctorClient{}, checkResult{Name: "grpc_client_setup", Status: statusFail}
		},
		bootstrapPeers: func(context.Context, bootstrapPeerValidation, time.Duration) []checkResult {
			return []checkResult{{Name: "bootstrap_peer_multiaddr", Status: statusSkip}}
		},
		dhtBootstrap: func(context.Context, bootstrapPeerValidation, time.Duration) checkResult {
			return checkResult{Name: "dht_bootstrap_reachability", Status: statusSkip}
		},
	})
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: localhost:8888
    auth_mode: insecure
`)

	output := runner.run(context.Background(), newDoctorTestCommand())

	require.Len(t, output.Results, 6)
	assert.Equal(t, "context_config", output.Results[0].Name)
	assert.Equal(t, "directory_api_tcp", output.Results[1].Name)
	assert.Equal(t, "grpc_client_setup", output.Results[2].Name)
	assert.Equal(t, "routing_list", output.Results[3].Name)
	assert.Equal(t, statusSkip, output.Results[3].Status)
	assert.Equal(t, "bootstrap_peer_multiaddr", output.Results[4].Name)
	assert.Equal(t, "dht_bootstrap_reachability", output.Results[5].Name)
}

func TestRunDoctorRoutesSuccessfulClientToRouting(t *testing.T) {
	resetDoctorTestState(t)

	routingCalled := false
	runner := newDoctorRunner(doctorDeps{
		directoryAPI: func(context.Context, string, time.Duration) checkResult {
			return checkResult{Name: "directory_api_tcp", Status: statusPass}
		},
		directoryClient: func(context.Context, *client.Config) (closer, checkResult) {
			return noopDoctorClient{}, checkResult{Name: "grpc_client_setup", Status: statusPass}
		},
		routingList: func(context.Context, closer, string, time.Duration) checkResult {
			routingCalled = true

			return checkResult{Name: "routing_list", Status: statusPass}
		},
		bootstrapPeers: func(context.Context, bootstrapPeerValidation, time.Duration) []checkResult {
			return nil
		},
		dhtBootstrap: func(context.Context, bootstrapPeerValidation, time.Duration) checkResult {
			return checkResult{Name: "dht_bootstrap_reachability", Status: statusSkip}
		},
	})
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: localhost:8888
    auth_mode: insecure
`)

	output := runner.run(context.Background(), newDoctorTestCommand())

	assert.True(t, routingCalled)
	assert.Equal(t, 4, output.Summary.Passed)
}

func TestRunDoctorReportsClientCloseWarning(t *testing.T) {
	resetDoctorTestState(t)

	runner := newDoctorRunner(doctorDeps{
		directoryAPI: func(context.Context, string, time.Duration) checkResult {
			return checkResult{Name: "directory_api_tcp", Status: statusPass}
		},
		directoryClient: func(context.Context, *client.Config) (closer, checkResult) {
			return closeErrDoctorClient{}, checkResult{Name: "grpc_client_setup", Status: statusPass}
		},
		routingList: func(context.Context, closer, string, time.Duration) checkResult {
			return checkResult{Name: "routing_list", Status: statusPass}
		},
		bootstrapPeers: func(context.Context, bootstrapPeerValidation, time.Duration) []checkResult {
			return nil
		},
		dhtBootstrap: func(context.Context, bootstrapPeerValidation, time.Duration) checkResult {
			return checkResult{Name: "dht_bootstrap_reachability", Status: statusSkip}
		},
	})
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	writeDirctlConfig(t, configHome, `
current_context: dev
contexts:
  dev:
    server_address: localhost:8888
    auth_mode: insecure
`)

	output := runner.run(context.Background(), newDoctorTestCommand())

	require.Len(t, output.Results, 6)
	assert.Equal(t, "grpc_client_close", output.Results[4].Name)
	assert.Equal(t, statusWarn, output.Results[4].Status)
	assert.Contains(t, output.Results[4].Details["error"], "close failed")
}

func TestRunCommandReturnsChecksFailed(t *testing.T) {
	resetDoctorTestState(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cmd := newDoctorTestCommand()
	cmd.SetOut(ioDiscard{})

	err := runCommand(cmd, nil)

	require.ErrorIs(t, err, ErrChecksFailed)
}

func TestCommandBypassesRootClientSetup(t *testing.T) {
	assert.NotNil(t, Command.PersistentPreRunE)
}

func resetDoctorTestState(t *testing.T) {
	t.Helper()

	opts.Timeout = time.Millisecond
	opts.BootstrapPeers = nil
	cliconfig.Context = ""
	cliconfig.Client = &client.Config{}

	keys := []string{
		"DIRECTORY_CLIENT_CONTEXT",
		"DIRECTORY_CLIENT_SERVER_ADDRESS",
		"DIRECTORY_CLIENT_TLS_SKIP_VERIFY",
		"DIRECTORY_CLIENT_TLS_CERT_FILE",
		"DIRECTORY_CLIENT_TLS_KEY_FILE",
		"DIRECTORY_CLIENT_TLS_CA_FILE",
		"DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH",
		"DIRECTORY_CLIENT_SPIFFE_TOKEN",
		"DIRECTORY_CLIENT_AUTH_MODE",
		"DIRECTORY_CLIENT_JWT_AUDIENCE",
		"DIRECTORY_CLIENT_OIDC_ISSUER",
		"DIRECTORY_CLIENT_OIDC_CLIENT_ID",
		"DIRECTORY_CLIENT_AUTH_TOKEN",
	}

	original := make(map[string]string, len(keys))

	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		original[key] = value
		present[key] = ok
		require.NoError(t, os.Unsetenv(key)) //nolint:usetesting // Need truly unset variables.
	}

	t.Cleanup(func() {
		opts.Timeout = defaultTimeout
		opts.BootstrapPeers = nil
		cliconfig.Context = ""
		cliconfig.Client = &client.DefaultConfig

		for _, key := range keys {
			if !present[key] {
				_ = os.Unsetenv(key) //nolint:usetesting // Restoring process env after manual unset.

				continue
			}

			_ = os.Setenv(key, original[key]) //nolint:usetesting // Restoring process env after manual unset.
		}
	})
}

func newDoctorTestCommand() *cobra.Command {
	root := &cobra.Command{Use: "dirctl"}
	root.PersistentFlags().StringVar(&cliconfig.Context, "context", "", "Directory client context name")
	root.PersistentFlags().StringVar(&cliconfig.Client.ServerAddress, "server-addr", cliconfig.Client.ServerAddress, "Directory Server API address")
	root.PersistentFlags().StringVar(&cliconfig.Client.AuthMode, "auth-mode", cliconfig.Client.AuthMode, "Authentication mode")
	root.PersistentFlags().StringVar(&cliconfig.Client.SpiffeSocketPath, "spiffe-socket-path", cliconfig.Client.SpiffeSocketPath, "Path to SPIFFE Workload API socket")
	root.PersistentFlags().StringVar(&cliconfig.Client.SpiffeToken, "spiffe-token", cliconfig.Client.SpiffeToken, "Path to JSON file containing SPIFFE token")
	root.PersistentFlags().StringVar(&cliconfig.Client.JWTAudience, "jwt-audience", cliconfig.Client.JWTAudience, "JWT audience")
	root.PersistentFlags().BoolVar(&cliconfig.Client.TlsSkipVerify, "tls-skip-verify", cliconfig.Client.TlsSkipVerify, "Skip TLS verification")
	root.PersistentFlags().StringVar(&cliconfig.Client.TlsCAFile, "tls-ca-file", cliconfig.Client.TlsCAFile, "Path to TLS CA file")
	root.PersistentFlags().StringVar(&cliconfig.Client.TlsCertFile, "tls-cert-file", cliconfig.Client.TlsCertFile, "Path to TLS certificate file")
	root.PersistentFlags().StringVar(&cliconfig.Client.TlsKeyFile, "tls-key-file", cliconfig.Client.TlsKeyFile, "Path to TLS key file")
	root.PersistentFlags().StringVar(&cliconfig.Client.OIDCIssuer, "oidc-issuer", cliconfig.Client.OIDCIssuer, "OIDC issuer URL")
	root.PersistentFlags().StringVar(&cliconfig.Client.OIDCClientID, "oidc-client-id", cliconfig.Client.OIDCClientID, "OIDC client ID")
	root.PersistentFlags().StringVar(&cliconfig.Client.AuthToken, "auth-token", cliconfig.Client.AuthToken, "Pre-issued Bearer token")

	cmd := &cobra.Command{Use: "doctor"}
	root.AddCommand(cmd)

	return cmd
}

func writeDirctlConfig(t *testing.T, configHome string, content string) {
	t.Helper()

	configPath := filepath.Join(configHome, "dirctl", "config.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o700))
	require.NoError(t, os.WriteFile(configPath, []byte(strings.TrimSpace(content)+"\n"), 0o600))
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) {
	return len(p), nil
}

type noopDoctorClient struct{}

func (noopDoctorClient) Close() error {
	return nil
}

type closeErrDoctorClient struct{}

func (closeErrDoctorClient) Close() error {
	return errors.New("close failed")
}
