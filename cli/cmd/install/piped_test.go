// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pipe returns a command whose stdin carries the given text.
func pipe(text string) *cobra.Command {
	cmd := &cobra.Command{Use: "install"}
	cmd.SetIn(strings.NewReader(text))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	return cmd
}

func TestReadPipedRefsTakesOnePerLine(t *testing.T) {
	refs, err := readPipedRefs(pipe("bafyone\nbafytwo\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"bafyone", "bafytwo"}, refs)
}

func TestReadPipedRefsIgnoresBlanksAndComments(t *testing.T) {
	// A hand-written list should be annotatable.
	refs, err := readPipedRefs(pipe("\n# the one we want\nbafyone\n   \n\t bafytwo \n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"bafyone", "bafytwo"}, refs)
}

func TestReadPipedRefsAcceptsNamesToo(t *testing.T) {
	// `dirctl search` prints CIDs, but `dirctl install list` prints names, and
	// resolving one costs the call install already makes.
	refs, err := readPipedRefs(pipe("cisco.com/agent:v1.0.0\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"cisco.com/agent:v1.0.0"}, refs)
}

func TestReadPipedRefsRefusesAnUnboundedList(t *testing.T) {
	// A pipe from a broad search must not rewrite every agent config before
	// anyone notices.
	_, err := readPipedRefs(pipe(strings.Repeat("bafyone\n", maxPipedRefs+1)))
	require.ErrorContains(t, err, "refusing to install more than")
}

func TestEmptyPipeIsNotAnError(t *testing.T) {
	refs, err := readPipedRefs(pipe(""))
	require.NoError(t, err)
	assert.Empty(t, refs)
}

func TestATerminalIsNotAPipe(t *testing.T) {
	// With no argument and a terminal, install prints help as it always has.
	cmd := &cobra.Command{Use: "install"}
	cmd.SetIn(os.Stdin)

	assert.Equal(t, !isTerminal(os.Stdin), hasPipedInput(cmd))

	// A redirect, a file, or /dev/null is a list, even an empty one.
	devNull, err := os.Open(os.DevNull)
	require.NoError(t, err)

	t.Cleanup(func() { _ = devNull.Close() })

	cmd.SetIn(devNull)
	assert.True(t, hasPipedInput(cmd))
}

func TestAPipedRunWillNotPrompt(t *testing.T) {
	resetOpts(t)

	// stdin carries the references, so a prompt would read a CID as the answer.
	opts.yes = false
	opts.dryRun = false

	_, err := confirmPiped()
	require.ErrorIs(t, err, errPipedNeedsYes)
	require.ErrorContains(t, err, "--yes")

	opts.yes = true
	ok, err := confirmPiped()
	require.NoError(t, err)
	assert.True(t, ok)

	opts.yes = false
	opts.dryRun = true
	ok, err = confirmPiped()
	require.NoError(t, err)
	assert.True(t, ok)
}
