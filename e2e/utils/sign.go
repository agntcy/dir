// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	"github.com/agntcy/dir/utils/cosign"
	"github.com/onsi/ginkgo/v2"
)

// Test constants for signature operations.
const (
	TestPassword = "testpassword"
)

// GenerateCosignKeyPair generates a cosign key pair in the specified directory.
// Helper function for signature testing.
func GenerateCosignKeyPair(dir string) {
	opts := &cosign.GenerateKeyPairOptions{
		Directory: dir,
		Password:  TestPassword,
	}

	if err := cosign.GenerateKeyPair(context.Background(), opts); err != nil {
		ginkgo.Fail("cosign generate-key-pair failed: " + err.Error())
	}
}
