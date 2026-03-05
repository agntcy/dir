// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/onsi/ginkgo/v2"
)

// Test constants for signature operations.
const (
	TestPassword = "testpassword"
)

// GenerateCosignKeyPair generates a cosign key pair in the specified directory.
// Helper function for signature testing.
func GenerateCosignKeyPair(dir string) {
	// Prepare cosign generate-key-pair command
	cmd := exec.CommandContext(context.Background(), "cosign", "generate-key-pair")
	cmd.Dir = dir

	cmd.Env = append(os.Environ(), "COSIGN_PASSWORD="+TestPassword)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("cosign generate-key-pair failed: %v\nOutput: %s", err, string(output)))
	}
}
