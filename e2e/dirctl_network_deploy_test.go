// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Using peer addresses from utils.constants

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a network multi peer deployment", func() {
	var cli *utils.CLI
	var cid string

	// Setup temp record file
	tempDir := os.Getenv("E2E_COMPILE_OUTPUT_DIR")
	if tempDir == "" {
		tempDir = os.TempDir()
	}
	tempPath := filepath.Join(tempDir, "record_v3_network_test.json")

	// Create directory and write V3 record data
	_ = os.MkdirAll(filepath.Dir(tempPath), 0o755)
	_ = os.WriteFile(tempPath, expectedRecordV3JSON, 0o600)

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	ginkgo.It("should push an record to peer 1", func() {
		cid = cli.Push(tempPath).OnServer(utils.Peer1Addr).ShouldSucceed()

		// Validate that the returned CID correctly represents the pushed data
		utils.LoadAndValidateCID(cid, tempPath)
	})

	ginkgo.It("should pull the record from peer 1", func() {
		cli.Pull(cid).OnServer(utils.Peer1Addr).ShouldSucceed()
	})

	ginkgo.It("should fail to pull the record from peer 2", func() {
		_ = cli.Pull(cid).OnServer(utils.Peer2Addr).ShouldFail()
	})

	ginkgo.It("should publish an record to the network on peer 1", func() {
		cli.Publish(cid).OnServer(utils.Peer1Addr).ShouldSucceed()

		// Wait at least 10 seconds to ensure the record is published.
		time.Sleep(15 * time.Second)
	})

	ginkgo.It("should fail publish an record to the network on peer 2 that does not store the record", func() {
		_ = cli.Publish(cid).OnServer(utils.Peer2Addr).ShouldFail()
	})

	ginkgo.It("should list by CID on all peers", func() {
		for _, addr := range utils.PeerAddrs {
			output := cli.List().WithCid(cid).OnServer(addr).ShouldSucceed()

			// Extract the Peer ID/hash from the output
			peerIndex := strings.Index(output, "Peer ")
			gomega.Expect(peerIndex).NotTo(gomega.Equal(-1), "Peer ID not found in output")

			peerIDStart := peerIndex + len("Peer ")
			peerIDEnd := strings.Index(output[peerIDStart:], "\n")
			gomega.Expect(peerIDEnd).NotTo(gomega.Equal(-1), "Newline not found after Peer ID")
			peerIDEnd += peerIDStart

			peerID := output[peerIDStart:peerIDEnd]

			// Build the expected output string
			expectedOutput := "Peer " + peerID + "\n" +
				"  CID: " + cid + "\n" +
				"  Labels: /skills/Natural Language Processing/Text Completion, /skills/Natural Language Processing/Problem Solving, /domains/research, /features/runtime/framework, /features/runtime/language"

			// Validate the output matches the expected format
			gomega.Expect(output).To(gomega.Equal(expectedOutput))
		}
	})

	ginkgo.It("should list by skill on all peers", func() {
		for _, addr := range utils.PeerAddrs {
			output := cli.List().WithSkill("/skills/Natural Language Processing").OnServer(addr).ShouldSucceed()

			// Extract the Peer ID/hash from the output
			peerIndex := strings.Index(output, "Peer ")
			gomega.Expect(peerIndex).NotTo(gomega.Equal(-1), "Peer ID not found in output")

			peerIDStart := peerIndex + len("Peer ")
			peerIDEnd := strings.Index(output[peerIDStart:], "\n")
			gomega.Expect(peerIDEnd).NotTo(gomega.Equal(-1), "Newline not found after Peer ID")
			peerIDEnd += peerIDStart

			peerID := output[peerIDStart:peerIDEnd]

			// Build the expected output string
			expectedOutput := "Peer " + peerID + "\n" +
				"  CID: " + cid + "\n" +
				"  Labels: /skills/Natural Language Processing/Text Completion, /skills/Natural Language Processing/Problem Solving, /domains/research, /features/runtime/framework, /features/runtime/language"

			// Validate the output matches the expected format
			gomega.Expect(output).To(gomega.Equal(expectedOutput))
		}
	})
})
