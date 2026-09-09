// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"testing"

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
