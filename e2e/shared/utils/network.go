// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os"
	"path/filepath"

	initcmd "github.com/agntcy/dir/cli/cmd/network/init"
	"github.com/onsi/gomega"
)

const (
	// PrivateKeyFileMode is the file permission for private key files (read/write for owner only).
	PrivateKeyFileMode = 0o0600
)

// GenerateNetworkKeyPair generates an ED25519 key pair for network tests.
// Returns the path to the private key file.
func GenerateNetworkKeyPair(tempDir string) string {
	// Generate OpenSSL-style ED25519 key
	_, privateKey, err := initcmd.GenerateED25519OpenSSLKey()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Write the private key to a temporary file
	keyPath := filepath.Join(tempDir, "test_key")
	err = os.WriteFile(keyPath, privateKey, PrivateKeyFileMode)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return keyPath
}

// SetupNetworkTestDir creates a temporary directory for network tests.
func SetupNetworkTestDir() (string, func()) {
	tempDir, err := os.MkdirTemp("", NetworkTestDirPrefix)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	cleanup := func() {
		err := os.RemoveAll(tempDir)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	}

	return tempDir, cleanup
}
