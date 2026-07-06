// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	zlog "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhaseFor(t *testing.T) {
	cases := []struct {
		line string
		want string
		ok   bool
	}{
		{"downloading model config.json", phaseDownload, true},
		{"86.64 MiB of 86.68 MiB (99%) downloaded", phaseDownload, true},
		{"226.08 KiB downloaded", phaseDownload, true},
		{`Serializing model to "spago_model.bin"...`, phaseEmbed, true}, // spago_model wins
		{"converting bert model", phaseConvert, true},
		{"model file already exists, skipping conversion", phaseEmbed, true},
		{"Done.", phaseEmbed, true},
		{"some unrelated line", "", false},
		{"", "", false},
	}

	for _, tc := range cases {
		got, ok := phaseFor(tc.line)
		assert.Equal(t, tc.ok, ok, "ok for %q", tc.line)
		assert.Equal(t, tc.want, got, "phase for %q", tc.line)
	}
}

// TestRunWithSpinnerCapturesNoise proves the wrapper intercepts stdout, stderr,
// and cybertron's global zerolog output (so none of it reaches the terminal) and
// surfaces it via the returned capture string, while propagating the action's
// error.
func TestRunWithSpinnerCapturesNoise(t *testing.T) {
	uiOut, err := os.CreateTemp(t.TempDir(), "ui-*")
	require.NoError(t, err)

	t.Cleanup(func() { _ = uiOut.Close() })

	wantErr := errors.New("provision boom")

	captured, err := runWithSpinner(context.Background(), uiOut, phaseDownload, phaseFor,
		func(_ context.Context) error {
			fmt.Fprintln(os.Stdout, "SECRET_STDOUT_NOISE")
			fmt.Fprintln(os.Stderr, "SECRET_STDERR_NOISE")
			zlog.Info().Msg("SECRET_ZEROLOG_NOISE")

			return wantErr
		})

	require.ErrorIs(t, err, wantErr)
	assert.Contains(t, captured, "SECRET_STDOUT_NOISE")
	assert.Contains(t, captured, "SECRET_STDERR_NOISE")
	assert.Contains(t, captured, "SECRET_ZEROLOG_NOISE")
}

// TestRunWithSpinnerSuccess confirms a nil-error action returns no error and
// that os.Stdout is restored afterward (regression guard for the redirect).
func TestRunWithSpinnerSuccess(t *testing.T) {
	uiOut, err := os.CreateTemp(t.TempDir(), "ui-*")
	require.NoError(t, err)

	t.Cleanup(func() { _ = uiOut.Close() })

	orig := os.Stdout

	_, err = runWithSpinner(context.Background(), uiOut, "Verifying setup…", nil,
		func(_ context.Context) error { return nil })

	require.NoError(t, err)
	assert.Same(t, orig, os.Stdout, "os.Stdout must be restored after the call")
}

// TestRunWithSpinnerRestoresOnPanic confirms a panic in the action still
// restores os.Stdout (and propagates the panic) rather than leaving the process
// with redirected output.
func TestRunWithSpinnerRestoresOnPanic(t *testing.T) {
	uiOut, err := os.CreateTemp(t.TempDir(), "ui-*")
	require.NoError(t, err)

	t.Cleanup(func() { _ = uiOut.Close() })

	orig := os.Stdout

	require.Panics(t, func() {
		_, _ = runWithSpinner(context.Background(), uiOut, phaseDownload, phaseFor,
			func(_ context.Context) error { panic("boom") })
	})

	assert.Same(t, orig, os.Stdout, "os.Stdout must be restored even when action panics")
}

func TestIndentLines(t *testing.T) {
	assert.Equal(t, "  a\n  b", indentLines("a\nb\n", "  "))
	assert.Equal(t, "  solo", indentLines("solo", "  "))
}
