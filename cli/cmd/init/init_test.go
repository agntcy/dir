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

func TestConfirm(t *testing.T) {
	cases := map[string]bool{
		"y\n":   true,
		"Y\n":   true,
		"yes\n": true,
		"n\n":   false,
		"\n":    false,
		"":      false, // EOF
	}
	for in, want := range cases {
		cmd, _ := newTestCmd(in)
		got, err := confirm(cmd, "Proceed?")
		require.NoError(t, err)
		assert.Equal(t, want, got, "input %q", in)
	}
}
