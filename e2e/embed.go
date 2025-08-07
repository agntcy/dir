// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

// Embedded test data files used across multiple test suites.
// This centralizes all test data to avoid duplication and ensure consistency.

//go:embed testdata/agent_v1.json
var expectedAgentV1JSON []byte

//go:embed testdata/agent_v2.json
var expectedAgentV2JSON []byte

//go:embed testdata/agent_v3.json
var expectedAgentV3JSON []byte
