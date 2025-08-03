// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"strings"

	"github.com/agntcy/dir/e2e/config"
	"github.com/agntcy/dir/e2e/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Using peer addresses from utils.constants

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a network multi peer deployment", func() {
	var cli *utils.CLI

	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}

		// Initialize CLI helper
		cli = utils.NewCLI()
	})

	// Test params
	var tempAgentCID string

	ginkgo.It("should push an agent to peer 1", func() {
		tempAgentCID = cli.Push("./testdata/agent.json").OnServer(utils.Peer1Addr).ShouldSucceed()

		// Validate that the returned CID correctly represents the pushed data
		utils.LoadAndValidateCID(tempAgentCID, "./testdata/agent.json")
	})

	ginkgo.It("should pull the agent from peer 1", func() {
		cli.Pull(tempAgentCID).OnServer(utils.Peer1Addr).ShouldSucceed()
	})

	ginkgo.It("should fail to pull the agent from peer 2", func() {
		_ = cli.Pull(tempAgentCID).OnServer(utils.Peer2Addr).ShouldFail()
	})

	ginkgo.It("should publish an agent to the network on peer 1", func() {
		cli.Publish(tempAgentCID).OnServer(utils.Peer1Addr).WithArgs("--network").ShouldSucceed()
	})

	ginkgo.It("should fail publish an agent to the network on peer 2 that does not store the agent", func() {
		_ = cli.Publish(tempAgentCID).OnServer(utils.Peer2Addr).WithArgs("--network").ShouldFail()
	})

	ginkgo.It("should list by CID on all peers", func() {
		for _, addr := range utils.PeerAddrs {
			output := cli.List().WithDigest(tempAgentCID).OnServer(addr).ShouldSucceed()

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
				"  CID: " + tempAgentCID + "\n" +
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
				"  CID: " + tempAgentCID + "\n" +
				"  Labels: /skills/Natural Language Processing/Text Completion, /skills/Natural Language Processing/Problem Solving, /domains/research, /features/runtime/framework, /features/runtime/language"

			// Validate the output matches the expected format
			gomega.Expect(output).To(gomega.Equal(expectedOutput))
		}
	})
})
