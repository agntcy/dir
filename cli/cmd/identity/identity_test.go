// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_HasExpectedSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command.Commands() {
		names[c.Name()] = true
	}

	assert.True(t, names["claim"])
	assert.True(t, names["status"])
	assert.True(t, names["resolve"])
}

func TestClaimCommand_FlagValidation(t *testing.T) {
	base := []string{"--record", "baguqeera-cid", "--role", "identity", "--subject", "did:web:acme.com"}

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "key alone", args: append(base, "--key", "key.pem")},
		{name: "key and cert", args: append(base, "--key", "key.pem", "--cert", "cert.pem")},
		{name: "cert without key", args: append(base, "--cert", "cert.pem"), wantErr: `"key" not set`},
		{name: "no key at all", args: base, wantErr: `"key" not set`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetFlags(t, claimCmd.Flags())

			require.NoError(t, claimCmd.ParseFlags(tt.args))

			err := claimCmd.ValidateRequiredFlags()
			if err == nil {
				err = claimCmd.ValidateFlagGroups()
			}

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

// pflag keeps a flag's Changed state across ParseFlags calls on the same set.
func resetFlags(t *testing.T, flags *pflag.FlagSet) {
	t.Helper()

	flags.VisitAll(func(f *pflag.Flag) {
		f.Changed = false
		require.NoError(t, f.Value.Set(f.DefValue))
	})
}

func TestLoadSigner_RequiresKey(t *testing.T) {
	_, err := loadSigner("", "")
	require.Error(t, err)
}

func TestLoadSigner_MissingKeyFile(t *testing.T) {
	_, err := loadSigner("/nonexistent/key.pem", "")
	require.Error(t, err)
}

func TestLoadSigner_MissingCertFile(t *testing.T) {
	_, err := loadSigner("/nonexistent/key.pem", "/nonexistent/cert.pem")
	require.Error(t, err)
}
