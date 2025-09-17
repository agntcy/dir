// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests for network commands", func() {
	var (
		cli         *utils.CLI
		tempDir     string
		tempKeyPath string
		cleanup     func()
	)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
		}

		// Initialize CLI helper
		cli = utils.NewCLI()

		// Setup test directory and generate network key
		tempDir, cleanup = utils.SetupNetworkTestDir()
		tempKeyPath = utils.GenerateNetworkKeyPair(tempDir)
	})

	ginkgo.AfterEach(func() {
		if cleanup != nil {
			cleanup()
		}
	})

	ginkgo.Context("info command", func() {
		ginkgo.It("should generate a peer ID from a valid ED25519 key", func() {
			output := cli.Network().Info(tempKeyPath).ShouldSucceed()

			// Verify that the output is not empty
			gomega.Expect(output).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should fail with non-existent key file", func() {
			_ = cli.Network().Info("non-existent-key-file").ShouldFail()
		})

		ginkgo.It("should fail with empty key path", func() {
			_ = cli.Network().Info("").ShouldFail()
		})
	})

	ginkgo.Context("init command", func() {
		ginkgo.It("should generate a new peer ID and save the key to specified output", func() {
			outputPath := filepath.Join(tempDir, "generated.key")

			// Generate new peer ID and key
			peerID := cli.Network().Init().WithOutput(outputPath).ShouldSucceed()

			// Verify that the output file exists with correct permissions
			gomega.Expect(outputPath).To(gomega.BeAnExistingFile())

			info, err := os.Stat(outputPath)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(info.Mode().Perm()).To(gomega.Equal(os.FileMode(0o0600)))

			// Verify that the peer ID is valid format
			gomega.Expect(peerID).NotTo(gomega.BeEmpty())
			gomega.Expect(peerID).To(gomega.HavePrefix("12D3"))

			// Verify that the generated key can be used with the info command
			infoOutput := cli.Network().Info(outputPath).ShouldSucceed()
			gomega.Expect(infoOutput).To(gomega.Equal(peerID))
		})

		ginkgo.It("should fail when output directory doesn't exist and cannot be created", func() {
			_ = cli.Network().Init().WithOutput("/nonexistent/directory/key.pem").ShouldFail()
		})
	})
})
