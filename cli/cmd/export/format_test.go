// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package export

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateExportFormatRejectsOASF(t *testing.T) {
	err := validateExportFormat("oasf")
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "pull")
}

func TestValidateExportFormatRejectsEmpty(t *testing.T) {
	err := validateExportFormat("")
	require.Error(t, err)
}

func TestValidateExportFormatAllowsKnownFormats(t *testing.T) {
	for _, f := range []string{"agent-skill", "skill", "a2a", "mcp-ghcopilot"} {
		assert.NoError(t, validateExportFormat(f), "format %q should be allowed", f)
	}
}
