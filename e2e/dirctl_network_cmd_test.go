// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	clicmd "github.com/agntcy/dir/cli/cmd"
	initcmd "github.com/agntcy/dir/cli/cmd/network/init"
	"github.com/agntcy/dir/e2e/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests for network commands", func() {
	var (
		tempDir     string
		tempKeyPath string
	)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		// Create temporary directory for test keys
		var err error
		tempDir, err = os.MkdirTemp("", "network-test")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Generate OpenSSL-style ED25519 key
		_, privateKey, err := initcmd.GenerateED25519OpenSSLKey()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Write the private key to a temporary file
		tempKeyPath = filepath.Join(tempDir, "test_key")
		err = os.WriteFile(tempKeyPath, privateKey, 0o0600)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		// Cleanup temporary directory
		err := os.RemoveAll(tempDir)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.Context("info command", func() {
		ginkgo.It("should generate a peer ID from a valid ED25519 key", func() {
			var outputBuffer bytes.Buffer

			infoCmd := clicmd.RootCmd
			infoCmd.SetOut(&outputBuffer)
			infoCmd.SetArgs([]string{
				"network",
				"info",
				tempKeyPath,
			})

			err := infoCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify that the output is not empty
			output := strings.TrimSpace(outputBuffer.String())
			gomega.Expect(output).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should fail with non-existent key file", func() {
			var outputBuffer bytes.Buffer

			infoCmd := clicmd.RootCmd
			infoCmd.SetOut(&outputBuffer)
			infoCmd.SetArgs([]string{
				"network",
				"info",
				"non-existent-key-file",
			})

			err := infoCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should fail with empty key path", func() {
			var outputBuffer bytes.Buffer

			infoCmd := clicmd.RootCmd
			infoCmd.SetOut(&outputBuffer)
			infoCmd.SetArgs([]string{
				"network",
				"info",
				"",
			})

			err := infoCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("init command", func() {
		ginkgo.It("should generate a new peer ID and save the key to specified output", func() {
			outputPath := filepath.Join(tempDir, "generated.key")
			var outputBuffer bytes.Buffer

			initCmd := clicmd.RootCmd
			initCmd.SetOut(&outputBuffer)
			initCmd.SetArgs([]string{
				"network",
				"init",
				"--output",
				outputPath,
			})

			err := initCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify that the output file exists
			_, err = os.Stat(outputPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify file permissions
			info, err := os.Stat(outputPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(info.Mode().Perm()).To(gomega.Equal(os.FileMode(0o0600)))

			// Verify that the output is not empty and is a valid peer ID
			output := strings.TrimSpace(outputBuffer.String())
			gomega.Expect(output).NotTo(gomega.BeEmpty())
			gomega.Expect(output).To(gomega.HavePrefix("12D3"))

			// Verify that the generated key can be used with the info command
			var infoBuffer bytes.Buffer
			infoCmd := clicmd.RootCmd
			infoCmd.SetOut(&infoBuffer)
			infoCmd.SetArgs([]string{
				"network",
				"info",
				outputPath,
			})

			err = infoCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify that info command returns the same peer ID
			infoOutput := strings.TrimSpace(infoBuffer.String())
			gomega.Expect(infoOutput).To(gomega.Equal(output))
		})

		ginkgo.It("should fail when output directory doesn't exist and cannot be created", func() {
			// Try to write to a location that should fail
			var outputBuffer bytes.Buffer

			initCmd := clicmd.RootCmd
			initCmd.SetOut(&outputBuffer)
			initCmd.SetArgs([]string{
				"network",
				"init",
				"--output",
				"/nonexistent/directory/key.pem",
			})

			err := initCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})
})
