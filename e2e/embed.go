// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

// Embedded test data files used across multiple test suites.
// This centralizes all test data to avoid duplication and ensure consistency.

//go:embed testdata/record_v1.json
var expectedRecordV1JSON []byte

//go:embed testdata/record_v2.json
var expectedRecordV2JSON []byte

//go:embed testdata/record_v3.json
var expectedRecordV3JSON []byte
