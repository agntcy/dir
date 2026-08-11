// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfirm(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{name: "short yes", input: "y\n", want: true},
		{name: "full yes", input: "yes\n", want: true},
		{name: "uppercase yes", input: "YES\n", want: true},
		{name: "no", input: "n\n", want: false},
		{name: "empty defaults to no", input: "\n", want: false},
		{name: "EOF", input: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetIn(strings.NewReader(test.input))

			var out bytes.Buffer
			cmd.SetOut(&out)

			confirmed, err := Confirm(cmd, "Proceed?")

			if test.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, "read confirmation")

				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.want, confirmed)
			assert.Contains(t, out.String(), "Proceed? [y/N]:")
		})
	}
}
