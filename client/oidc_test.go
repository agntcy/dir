// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadOAuthResponseBodyWithinLimit(t *testing.T) {
	body := strings.NewReader(`{"access_token":"ok"}`)
	data, err := readOAuthResponseBody(body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"access_token":"ok"}`, string(data))
}

func TestReadOAuthResponseBodyExceedsLimit(t *testing.T) {
	oversized := strings.Repeat("a", maxOAuthResponseBodySize+1)
	body := strings.NewReader(oversized)

	data, err := readOAuthResponseBody(body)
	require.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "oauth response body exceeds")
}
