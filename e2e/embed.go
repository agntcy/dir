// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

// Embedded test data files used across multiple test suites.
// This centralizes all test data to avoid duplication and ensure consistency.

//go:embed testdata/record_v1alpha0.json
var expectedRecordV1Alpha0JSON []byte

//go:embed testdata/record_v1alpha1.json
var expectedRecordV1Alpha1JSON []byte
