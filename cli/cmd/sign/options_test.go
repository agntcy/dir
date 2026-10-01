// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sign

import (
	"os"
	"path/filepath"
	"testing"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	cliconfig "github.com/agntcy/dir/cli/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sigstoreConfig = `
current_context: corp
contexts:
  public:
    server_address: public.example.com:443
  corp:
    server_address: corp.example.com:443
    sigstore:
      fulcio_url: https://fulcio.corp.example
      rekor_url: https://rekor.corp.example
      timestamp_url: https://tsa.corp.example/api/v1/timestamp
      skip_tlog: true
      oidc_provider_url: https://idp.corp.example
      oidc_client_id: corp-sigstore
`

var sigstoreEnvVars = []string{
	"DIRECTORY_CLIENT_CONTEXT",
	"DIRECTORY_CLIENT_SIGSTORE_FULCIO_URL",
	"DIRECTORY_CLIENT_SIGSTORE_REKOR_URL",
	"DIRECTORY_CLIENT_SIGSTORE_TIMESTAMP_URL",
	"DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG",
	"DIRECTORY_CLIENT_SIGSTORE_OIDC_PROVIDER_URL",
	"DIRECTORY_CLIENT_SIGSTORE_OIDC_CLIENT_ID",
}

// setupSigstoreConfig points the default config path at a file holding
// sigstoreConfig and clears every environment variable that could override it.
func setupSigstoreConfig(t *testing.T) {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	configDir := filepath.Join(configHome, "dirctl")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(sigstoreConfig), 0o600))

	for _, key := range sigstoreEnvVars {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key)) //nolint:usetesting // Need truly unset variables.
	}

	originalContext := cliconfig.Context
	cliconfig.Context = ""

	t.Cleanup(func() { cliconfig.Context = originalContext })
}

// newSignCommand returns a command carrying the signing flags, parsed from args.
func newSignCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()

	original := *opts

	t.Cleanup(func() { *opts = original })

	cmd := &cobra.Command{Use: "sign"}
	AddSigningFlags(cmd.Flags())
	require.NoError(t, cmd.Flags().Parse(args))

	return cmd
}

func TestResolveOptionsUsesPublicDefaultsWithoutSigstoreSection(t *testing.T) {
	setupSigstoreConfig(t)

	cliconfig.Context = "public"

	resolved, err := ResolveOptions(newSignCommand(t))

	require.NoError(t, err)
	assert.Equal(t, signv1.DefaultFulcioURL, resolved.FulcioURL)
	assert.Equal(t, signv1.DefaultRekorURL, resolved.RekorURL)
	assert.Equal(t, signv1.DefaultTimestampURL, resolved.TimestampURL)
	assert.Equal(t, signv1.DefaultOIDCProviderURL, resolved.OIDCProviderURL)
	assert.Equal(t, signv1.DefaultOIDCClientID, resolved.OIDCClientID)
	assert.False(t, resolved.SkipTlog)
}

func TestResolveOptionsUsesContextSigstoreSection(t *testing.T) {
	setupSigstoreConfig(t)

	resolved, err := ResolveOptions(newSignCommand(t))

	require.NoError(t, err)
	assert.Equal(t, "https://fulcio.corp.example", resolved.FulcioURL)
	assert.Equal(t, "https://rekor.corp.example", resolved.RekorURL)
	assert.Equal(t, "https://tsa.corp.example/api/v1/timestamp", resolved.TimestampURL)
	assert.Equal(t, "https://idp.corp.example", resolved.OIDCProviderURL)
	assert.Equal(t, "corp-sigstore", resolved.OIDCClientID)
	assert.True(t, resolved.SkipTlog)
}

func TestResolveOptionsPrecedence(t *testing.T) {
	setupSigstoreConfig(t)
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_FULCIO_URL", "https://fulcio.env.example")
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_REKOR_URL", "https://rekor.env.example")

	resolved, err := ResolveOptions(newSignCommand(t,
		"--rekor-url", "https://rekor.flag.example",
		"--skip-tlog=false",
	))

	require.NoError(t, err)
	assert.Equal(t, "https://fulcio.env.example", resolved.FulcioURL, "env overrides config")
	assert.Equal(t, "https://rekor.flag.example", resolved.RekorURL, "flag overrides env")
	assert.False(t, resolved.SkipTlog, "an explicit false flag overrides config")
	assert.Equal(t, "https://idp.corp.example", resolved.OIDCProviderURL, "config overrides default")
}

func TestResolveOptionsLeavesPackageOptionsUntouched(t *testing.T) {
	setupSigstoreConfig(t)

	cmd := newSignCommand(t)

	_, err := ResolveOptions(cmd)

	require.NoError(t, err)
	assert.Equal(t, signv1.DefaultFulcioURL, opts.FulcioURL)
	assert.False(t, opts.SkipTlog)
}

func TestResolveOptionsSkipsSigstoreForKeySigning(t *testing.T) {
	setupSigstoreConfig(t)
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG", "not-a-bool")

	resolved, err := ResolveOptions(newSignCommand(t, "--key", "cosign.key"))

	require.NoError(t, err)
	assert.Equal(t, "cosign.key", resolved.Key)
}

func TestResolveOptionsRejectsInvalidEnvironment(t *testing.T) {
	setupSigstoreConfig(t)
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG", "not-a-bool")

	_, err := ResolveOptions(newSignCommand(t))

	require.ErrorContains(t, err, "DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG")
}
