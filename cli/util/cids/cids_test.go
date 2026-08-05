// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cids

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFromStdinJSONArray(t *testing.T) {
	cids, err := ReadFromStdin(strings.NewReader("[\"bafy1\", \"bafy2\"]\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"bafy1", "bafy2"}, cids)
}

func TestReadFromStdinLineDelimited(t *testing.T) {
	cids, err := ReadFromStdin(strings.NewReader("bafy1\n  bafy2  \n\nbafy3\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"bafy1", "bafy2", "bafy3"}, cids)
}

func TestReadFromStdinEmpty(t *testing.T) {
	cids, err := ReadFromStdin(strings.NewReader("   \n"))
	require.NoError(t, err)
	assert.Empty(t, cids)
}

func TestReadFromStdinInvalidJSON(t *testing.T) {
	_, err := ReadFromStdin(strings.NewReader(`[{"cid": "bafy1"}]`))
	require.Error(t, err)
}

func TestDeduplicate(t *testing.T) {
	assert.Equal(t,
		[]string{"bafy1", "bafy2"},
		Deduplicate([]string{" bafy1 ", "bafy2", "bafy1", "", "  "}),
	)
}

func TestCollectMergesArgsAndStdin(t *testing.T) {
	cids, err := Collect([]string{"bafy1", "bafy2"}, strings.NewReader("bafy2\nbafy3\n"), true)
	require.NoError(t, err)
	assert.Equal(t, []string{"bafy1", "bafy2", "bafy3"}, cids)
}

func TestCollectIgnoresStdinWhenNotRequested(t *testing.T) {
	cids, err := Collect([]string{"bafy1"}, strings.NewReader("bafy2\n"), false)
	require.NoError(t, err)
	assert.Equal(t, []string{"bafy1"}, cids)
}

func TestCollectRequiresAtLeastOneCID(t *testing.T) {
	_, err := Collect(nil, strings.NewReader("\n"), true)
	require.ErrorContains(t, err, "at least one CID is required")
}

func TestArgs(t *testing.T) {
	fromStdin := false
	cmd := &cobra.Command{Use: "delete"}
	validate := Args(&fromStdin)

	require.Error(t, validate(cmd, nil))
	require.NoError(t, validate(cmd, []string{"bafy1"}))
	require.NoError(t, validate(cmd, []string{"bafy1", "bafy2"}))

	fromStdin = true

	require.NoError(t, validate(cmd, nil))
	require.Error(t, validate(cmd, []string{"bafy1"}))
}
