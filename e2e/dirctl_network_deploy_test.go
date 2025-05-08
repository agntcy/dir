// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"strings"

	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/agntcy/dir/e2e/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
)

const (
	Peer1Addr = "0.0.0.0:8890"
	Peer2Addr = "0.0.0.0:8891"
	Peer3Addr = "0.0.0.0:8892"
)

var peerAddrs = []string{Peer1Addr, Peer2Addr, Peer3Addr}

var _ = ginkgo.Describe("Running dirctl end-to-end tests using a network multi peer deployment", func() {
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}
	})

	// Test params
	var tempAgentDigest string

	ginkgo.It("should push an agent to peer 1", func() {
		var outputBuffer bytes.Buffer

		pushCmd := clicmd.RootCmd
		pushCmd.SetOut(&outputBuffer)
		pushCmd.SetArgs([]string{
			"push",
			"./testdata/agent.json",
			"--server-addr",
			Peer1Addr,
		})

		err := pushCmd.Execute()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		tempAgentDigest = outputBuffer.String()

		// Ensure the digest is valid
		_, err = digest.Parse(tempAgentDigest)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.It("should pull the agent from peer 1", func() {
		var outputBuffer bytes.Buffer

		pullCmd := clicmd.RootCmd
		pullCmd.SetOut(&outputBuffer)
		pullCmd.SetArgs([]string{
			"pull",
			tempAgentDigest,
			"--server-addr",
			Peer1Addr,
		})

		err := pullCmd.Execute()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.It("should fail to pull the agent from peer 2", func() {
		var outputBuffer bytes.Buffer

		pullCmd := clicmd.RootCmd
		pullCmd.SetOut(&outputBuffer)
		pullCmd.SetArgs([]string{
			"pull",
			tempAgentDigest,
			"--server-addr",
			Peer2Addr,
		})

		err := pullCmd.Execute()
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should publish an agent to the network on peer 1", func() {
		var outputBuffer bytes.Buffer

		publishCmd := clicmd.RootCmd
		publishCmd.SetOut(&outputBuffer)
		publishCmd.SetArgs([]string{
			"publish",
			tempAgentDigest,
			"--server-addr",
			Peer1Addr,
			"--network",
		})

		err := publishCmd.Execute()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.It("should fail publish an agent to the network on peer 2 that does not store the agent", func() {
		var outputBuffer bytes.Buffer

		publishCmd := clicmd.RootCmd
		publishCmd.SetOut(&outputBuffer)
		publishCmd.SetArgs([]string{
			"publish",
			tempAgentDigest,
			"--server-addr",
			Peer2Addr,
			"--network",
		})

		err := publishCmd.Execute()
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should list by digest on all peers", func() {
		for _, addr := range peerAddrs {
			var outputBuffer bytes.Buffer

			listCmd := clicmd.RootCmd
			listCmd.SetOut(&outputBuffer)
			listCmd.SetArgs([]string{
				"list",
				"--digest",
				tempAgentDigest,
				"--server-addr",
				addr,
			})

			err := listCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate the output contains the expected digest
			output := strings.TrimSpace(outputBuffer.String()) // Trim trailing whitespace and newlines

			// Extract the Peer ID/hash from the output
			peerIDStart := strings.Index(output, "Peer ") + len("Peer ")
			peerIDEnd := strings.Index(output[peerIDStart:], "\n") + peerIDStart
			gomega.Expect(peerIDStart).NotTo(gomega.Equal(-1), "Peer ID not found in output")
			peerID := output[peerIDStart:peerIDEnd]

			// Build the expected output string
			expectedOutput := "Peer " + peerID + "\n" +
				"  Digest: " + tempAgentDigest + "\n" +
				"  Labels: /skills/Natural Language Processing/Text Completion, /skills/Natural Language Processing/Problem Solving, /domains/research, /features/runtime/framework, /features/runtime/language"

			// Validate the output matches the expected format
			gomega.Expect(output).To(gomega.Equal(expectedOutput))
		}
	})

	ginkgo.It("should list by skill on all peers", func() {
		for _, addr := range peerAddrs {
			var outputBuffer bytes.Buffer

			listCmd := clicmd.RootCmd
			listCmd.SetOut(&outputBuffer)
			listCmd.SetArgs([]string{
				"list",
				"\"/skills/Natural Language Processing\"",
				"--server-addr",
				addr,
			})

			err := listCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Validate the output contains the expected digest
			output := strings.TrimSpace(outputBuffer.String()) // Trim trailing whitespace and newlines

			// Extract the Peer ID/hash from the output
			peerIDStart := strings.Index(output, "Peer ") + len("Peer ")
			peerIDEnd := strings.Index(output[peerIDStart:], "\n") + peerIDStart
			gomega.Expect(peerIDStart).NotTo(gomega.Equal(-1), "Peer ID not found in output")
			peerID := output[peerIDStart:peerIDEnd]

			// Build the expected output string
			expectedOutput := "Peer " + peerID + "\n" +
				"  Digest: " + tempAgentDigest + "\n" +
				"  Labels: /skills/Natural Language Processing/Text Completion, /skills/Natural Language Processing/Problem Solving, /domains/research, /features/runtime/framework, /features/runtime/language"

			// Validate the output matches the expected format
			gomega.Expect(output).To(gomega.Equal(expectedOutput))
		}
	})
})
