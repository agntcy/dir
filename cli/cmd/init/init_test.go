// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package init

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCmd(stdin string) (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetIn(strings.NewReader(stdin))

	return cmd, out
}

func TestConfirmDefaultNo(t *testing.T) {
	cases := map[string]bool{
		"y\n":   true,
		"Y\n":   true,
		"yes\n": true,
		"n\n":   false,
		"\n":    false, // Enter → default (No)
		"":      false, // EOF
	}
	for in, want := range cases {
		cmd, _ := newTestCmd(in)
		got, err := confirm(cmd, "Proceed?", false)
		require.NoError(t, err)
		assert.Equal(t, want, got, "input %q", in)
	}
}

func TestConfirmDefaultYes(t *testing.T) {
	cases := map[string]bool{
		"y\n":   true,
		"yes\n": true,
		"n\n":   false,
		"no\n":  false,
		"\n":    true,  // Enter → default (Yes)
		"":      false, // EOF: never assume yes without a keystroke
	}
	for in, want := range cases {
		cmd, _ := newTestCmd(in)
		got, err := confirm(cmd, "Proceed?", true)
		require.NoError(t, err)
		assert.Equal(t, want, got, "input %q", in)
	}
}
