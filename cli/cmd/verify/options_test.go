// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

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
      tuf_mirror_url: https://tuf.corp.example
      trusted_root_path: /etc/dirctl/trusted_root.json
      ignore_tlog: true
      ignore_tsa: true
      ignore_sct: true
`

var sigstoreEnvVars = []string{
	"DIRECTORY_CLIENT_CONTEXT",
	"DIRECTORY_CLIENT_SIGSTORE_TUF_MIRROR_URL",
	"DIRECTORY_CLIENT_SIGSTORE_TRUSTED_ROOT_PATH",
	"DIRECTORY_CLIENT_SIGSTORE_IGNORE_TLOG",
	"DIRECTORY_CLIENT_SIGSTORE_IGNORE_TSA",
	"DIRECTORY_CLIENT_SIGSTORE_IGNORE_SCT",
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

// newVerifyCommand returns a command carrying the OIDC trust flags, parsed from args.
func newVerifyCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()

	original := opts

	t.Cleanup(func() { opts = original })

	cmd := &cobra.Command{Use: "verify"}
	addOIDCOptionFlags(cmd.Flags())
	require.NoError(t, cmd.Flags().Parse(args))

	return cmd
}

func TestResolveOIDCOptionsUsesDefaultsWithoutSigstoreSection(t *testing.T) {
	setupSigstoreConfig(t)

	cliconfig.Context = "public"

	resolved, err := resolveOIDCOptions(newVerifyCommand(t))

	require.NoError(t, err)
	assert.Equal(t, signv1.DefaultVerifyOptionsOIDC.GetTufMirrorUrl(), resolved.GetTufMirrorUrl())
	assert.Empty(t, resolved.GetTrustedRootPath())
	assert.False(t, resolved.GetIgnoreTlog())
	assert.False(t, resolved.GetIgnoreTsa())
	assert.False(t, resolved.GetIgnoreSct())
}

func TestResolveOIDCOptionsUsesContextSigstoreSection(t *testing.T) {
	setupSigstoreConfig(t)

	resolved, err := resolveOIDCOptions(newVerifyCommand(t))

	require.NoError(t, err)
	assert.Equal(t, "https://tuf.corp.example", resolved.GetTufMirrorUrl())
	assert.Equal(t, "/etc/dirctl/trusted_root.json", resolved.GetTrustedRootPath())
	assert.True(t, resolved.GetIgnoreTlog())
	assert.True(t, resolved.GetIgnoreTsa())
	assert.True(t, resolved.GetIgnoreSct())
}

func TestResolveOIDCOptionsPrecedence(t *testing.T) {
	setupSigstoreConfig(t)
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_TRUSTED_ROOT_PATH", "/env/trusted_root.json")
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_TUF_MIRROR_URL", "https://tuf.env.example")
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_IGNORE_TSA", "false")

	resolved, err := resolveOIDCOptions(newVerifyCommand(t,
		"--tuf-mirror-url", "https://tuf.flag.example",
		"--ignore-sct=false",
	))

	require.NoError(t, err)
	assert.Equal(t, "/env/trusted_root.json", resolved.GetTrustedRootPath(), "env overrides config")
	assert.Equal(t, "https://tuf.flag.example", resolved.GetTufMirrorUrl(), "flag overrides env")
	assert.False(t, resolved.GetIgnoreTsa(), "an env false overrides config")
	assert.False(t, resolved.GetIgnoreSct(), "an explicit false flag overrides config")
	assert.True(t, resolved.GetIgnoreTlog(), "config overrides default")
}
