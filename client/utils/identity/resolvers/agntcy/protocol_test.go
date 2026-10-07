// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agntcy

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// Shared wire vectors guard the draft contract in Directory and Identity.
func TestWireContract(t *testing.T) {
	requestJSON, err := os.ReadFile("testdata/request.json")
	require.NoError(t, err)

	var request VerificationRequest
	require.NoError(t, json.Unmarshal(requestJSON, &request))
	encoded, err := json.Marshal(request)
	require.NoError(t, err)
	require.Equal(t, bytes.TrimSpace(requestJSON), encoded)

	resultJSON, err := os.ReadFile("testdata/result.json")
	require.NoError(t, err)

	var result VerificationResult
	require.NoError(t, json.Unmarshal(resultJSON, &result))
	encodedResult, err := json.Marshal(result)
	require.NoError(t, err)
	require.Equal(t, bytes.TrimSpace(resultJSON), encodedResult)
	require.Equal(t, ProtocolVersion, result.Version)
	require.Equal(t, Profile, result.Profile)
	require.Equal(t, SubjectKeyPolicy, result.PolicyVersion)
	require.Equal(t, DigestRequest(encoded), result.RequestDigest)
	require.True(t, result.Checks.Identity && result.Checks.Badge)
}
