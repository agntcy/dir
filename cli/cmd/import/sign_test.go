// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	signcmd "github.com/agntcy/dir/cli/cmd/sign"
	cliconfig "github.com/agntcy/dir/cli/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sigstoreConfig = `
current_context: corp
contexts:
  corp:
    server_address: corp.example.com:443
    sigstore:
      fulcio_url: https://fulcio.corp.example
      skip_tlog: true
`

// recordingSigner captures the sign request instead of sending it.
type recordingSigner struct {
	req *signv1.SignRequest
}

func (r *recordingSigner) Sign(_ context.Context, req *signv1.SignRequest) (*signv1.SignResponse, error) {
	r.req = req

	return &signv1.SignResponse{}, nil
}

// newSigningCommand points the default config path at sigstoreConfig and
// returns a command carrying the signing flags, parsed from args.
func newSigningCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()

	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("DIRECTORY_CLIENT_CONTEXT", "")
	require.NoError(t, os.Unsetenv("DIRECTORY_CLIENT_CONTEXT")) //nolint:usetesting // An empty value would select no context.
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_FULCIO_URL", "")
	t.Setenv("DIRECTORY_CLIENT_SIGSTORE_SKIP_TLOG", "")

	configDir := filepath.Join(configHome, "dirctl")
	require.NoError(t, os.MkdirAll(configDir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(sigstoreConfig), 0o600))

	originalContext := cliconfig.Context
	cliconfig.Context = ""

	t.Cleanup(func() {
		cliconfig.Context = originalContext
		// Registering the flags again resets the shared signing options to their defaults.
		signcmd.AddSigningFlags(pflag.NewFlagSet("reset", pflag.ContinueOnError))
	})

	cmd := &cobra.Command{Use: "import"}
	signcmd.AddSigningFlags(cmd.Flags())
	require.NoError(t, cmd.Flags().Parse(args))

	return cmd
}

func TestConfigureSigningLeavesSignFuncUnsetWithoutSignFlag(t *testing.T) {
	cmd := newSigningCommand(t)

	original := opts.SignFunc

	t.Cleanup(func() { opts.SignFunc = original })

	opts.SignFunc = nil

	require.NoError(t, configureSigning(cmd, &recordingSigner{}))
	assert.Nil(t, opts.SignFunc)
}

func TestConfigureSigningSendsResolvedSigstoreOptions(t *testing.T) {
	cmd := newSigningCommand(t, "--oidc-token", "id-token")

	originalSign, originalSignFunc := opts.Sign, opts.SignFunc
	opts.Sign = true

	t.Cleanup(func() { opts.Sign, opts.SignFunc = originalSign, originalSignFunc })

	signer := &recordingSigner{}

	require.NoError(t, configureSigning(cmd, signer))
	require.NotNil(t, opts.SignFunc)
	require.NoError(t, opts.SignFunc(t.Context(), "bafytest"))

	oidcOptions := signer.req.GetProvider().GetOidc().GetOptions()
	assert.Equal(t, "https://fulcio.corp.example", oidcOptions.GetFulcioUrl())
	assert.True(t, oidcOptions.GetSkipTlog())
}
