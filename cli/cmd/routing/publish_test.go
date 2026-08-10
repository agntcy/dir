// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setPublishOptionsForTest(
	t *testing.T, fromStdin bool, all bool, yes bool,
) {
	t.Helper()

	previous := publishOpts
	publishOpts.FromStdin = fromStdin
	publishOpts.All = all
	publishOpts.Yes = yes

	t.Cleanup(func() {
		publishOpts = previous
	})
}

func TestValidatePublishArgs(t *testing.T) {
	tests := []struct {
		name            string
		fromStdin       bool
		all             bool
		yes             bool
		args            []string
		wantErr         bool
		wantErrContains string
	}{
		{
			name: "all without CIDs",
			all:  true,
		},
		{
			name:            "all with stdin",
			fromStdin:       true,
			all:             true,
			wantErr:         true,
			wantErrContains: "--all cannot be used with --stdin",
		},
		{
			name:            "all with CID",
			all:             true,
			args:            []string{"test-cid"},
			wantErr:         true,
			wantErrContains: "--all cannot be used with CID arguments",
		},
		{
			name:            "yes without all",
			yes:             true,
			args:            []string{"test-cid"},
			wantErr:         true,
			wantErrContains: "--yes can only be used with --all",
		},
		{
			name:    "single CID",
			args:    []string{"test-cid"},
			wantErr: false,
		},
		{
			name:      "stdin without arguments",
			fromStdin: true,
			wantErr:   false,
		},
		{
			name:      "stdin with argument",
			fromStdin: true,
			args:      []string{"test-cid"},
			wantErr:   true,
		},
		{
			name:    "missing input",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setPublishOptionsForTest(
				t, test.fromStdin, test.all, test.yes)

			err := validatePublishArgs(
				&cobra.Command{Use: "publish"}, test.args)

			if test.wantErr {
				require.Error(t, err)

				if test.wantErrContains != "" {
					require.ErrorContains(t, err, test.wantErrContains)
				}

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestConfirmPublishAllIfNeeded(t *testing.T) {
	t.Run("non-terminal requires yes", func(t *testing.T) {
		setPublishOptionsForTest(t, false, true, false)

		cmd := &cobra.Command{}
		cmd.SetIn(strings.NewReader("y\n"))

		confirmed, err := confirmPublishAllIfNeeded(cmd)

		require.Error(t, err)
		assert.False(t, confirmed)
		require.ErrorContains(t, err, "pass --yes")
	})

	t.Run("yes skips prompt on non-terminal input", func(t *testing.T) {
		setPublishOptionsForTest(t, false, true, true)

		cmd := &cobra.Command{}
		cmd.SetIn(strings.NewReader(""))

		confirmed, err := confirmPublishAllIfNeeded(cmd)

		require.NoError(t, err)
		assert.True(t, confirmed)
	})
}

func TestConfirmPublishAll(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{
			name:  "short yes",
			input: "y\n",
			want:  true,
		},
		{
			name:  "full uppercase yes",
			input: "YES\n",
			want:  true,
		},
		{
			name:  "no",
			input: "n\n",
			want:  false,
		},
		{
			name:  "empty response defaults to no",
			input: "\n",
			want:  false,
		},
		{
			name:    "EOF",
			input:   "",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(test.input))

			var output bytes.Buffer
			cmd.SetOut(&output)

			confirmed, err := confirmPublishAll(cmd)

			if test.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, "read confirmation")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, confirmed)
			assert.Contains(t, output.String(), publishAllPrompt)
			assert.Contains(t, output.String(), "[y/N]")
		})
	}
}

func TestNormalizePublishError(t *testing.T) {
	t.Run("background retry", func(t *testing.T) {
		err := normalizePublishError(
			errors.New("failed to announce object: temporary failure"))

		assert.EqualError(
			t,
			err,
			"failed to announce object, it will be retried in the background on the API server",
		)
	})

	t.Run("other error", func(t *testing.T) {
		err := normalizePublishError(errors.New("permission denied"))

		assert.EqualError(t, err, "failed to publish: permission denied")
	})
}
