// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	_ "embed"
	"strings"
	"time"

	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/agntcy/dir/e2e/config"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	// Kubernetes internal service address for peer1.
	Peer1InternalAddr = "agntcy-dir-apiserver.peer1.svc.cluster.local:8888"
)

//go:embed testdata/agent_v2.json
var expectedAgentV2JSON []byte

var _ = ginkgo.Describe("Running dirctl end-to-end tests for sync commands", func() {
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeNetwork {
			ginkgo.Skip("Skipping test, not in network mode")
		}
	})

	var syncID string

	ginkgo.Context("create command", func() {
		ginkgo.It("should accept valid remote URL format", func() {
			var outputBuffer bytes.Buffer

			createCmd := clicmd.RootCmd
			createCmd.SetOut(&outputBuffer)
			createCmd.SetArgs([]string{
				"sync",
				"create",
				"https://directory.example.com",
				"--server-addr",
				Peer1Addr,
			})

			err := createCmd.Execute()
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("required"))
			}

			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring("Sync created with ID: "))
			syncID = strings.TrimPrefix(outputBuffer.String(), "Sync created with ID: ")
		})
	})

	ginkgo.Context("list command", func() {
		ginkgo.It("should execute without arguments and return a list with the created sync", func() {
			var outputBuffer bytes.Buffer

			listCmd := clicmd.RootCmd
			listCmd.SetOut(&outputBuffer)
			listCmd.SetArgs([]string{
				"sync",
				"list",
				"--server-addr",
				Peer1Addr,
			})

			err := listCmd.Execute()
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("argument"))
			}

			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring(syncID))
			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring("https://directory.example.com"))
			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring("PENDING"))
		})
	})

	ginkgo.Context("status command", func() {
		ginkgo.It("should accept a sync ID argument and return the sync status", func() {
			var outputBuffer bytes.Buffer

			statusCmd := clicmd.RootCmd
			statusCmd.SetOut(&outputBuffer)
			statusCmd.SetArgs([]string{
				"sync",
				"status",
				syncID,
				"--server-addr",
				Peer1Addr,
			})

			err := statusCmd.Execute()
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("required"))
			}

			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring(syncID))
			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring("PENDING"))
		})
	})

	ginkgo.Context("delete command", func() {
		ginkgo.It("should accept a sync ID argument and delete the sync", func() {
			var outputBuffer bytes.Buffer

			deleteCmd := clicmd.RootCmd
			deleteCmd.SetOut(&outputBuffer)
			deleteCmd.SetArgs([]string{
				"sync",
				"delete",
				syncID,
				"--server-addr",
				Peer1Addr,
			})

			err := deleteCmd.Execute()
			// Command may fail due to network/auth issues, but argument parsing should work
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("required"))
			}
		})

		ginkgo.It("should return deleted status", func() {
			var outputBuffer bytes.Buffer

			listCmd := clicmd.RootCmd
			listCmd.SetOut(&outputBuffer)
			listCmd.SetArgs([]string{
				"sync",
				"list",
				"--server-addr",
				Peer1Addr,
			})

			err := listCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring("DELETE"))
		})
	})

	ginkgo.Context("sync functionality", func() {
		var agentCID string

		ginkgo.It("should push agent_v2.json to peer 1", func() {
			var outputBuffer bytes.Buffer

			pushCmd := clicmd.RootCmd
			pushCmd.SetOut(&outputBuffer)
			pushCmd.SetArgs([]string{
				"push",
				"./testdata/agent_v2.json",
				"--server-addr",
				Peer1Addr,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			agentCID = strings.TrimSpace(outputBuffer.String())
		})

		ginkgo.It("should fail to pull agent_v2.json from peer 2", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				agentCID,
				"--server-addr",
				Peer2Addr,
			})

			err := pullCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})

		ginkgo.It("should create sync from peer 1 to peer 2", func() {
			var outputBuffer bytes.Buffer

			createCmd := clicmd.RootCmd
			createCmd.SetOut(&outputBuffer)
			createCmd.SetArgs([]string{
				"sync",
				"create",
				Peer1InternalAddr,
				"--server-addr",
				Peer2Addr,
			})

			err := createCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			output := strings.TrimSpace(outputBuffer.String())
			gomega.Expect(output).To(gomega.ContainSubstring("Sync created with ID: "))
			syncID = strings.TrimPrefix(outputBuffer.String(), "Sync created with ID: ")
		})

		ginkgo.It("should list the sync", func() {
			var outputBuffer bytes.Buffer

			listCmd := clicmd.RootCmd
			listCmd.SetOut(&outputBuffer)
			listCmd.SetArgs([]string{
				"sync",
				"list",
			})

			err := listCmd.Execute()
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("argument"))
			}

			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring(syncID))
			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring(Peer1InternalAddr))
			gomega.Expect(outputBuffer.String()).To(gomega.ContainSubstring("PENDING"))
		})

		// Wait for sync to complete
		ginkgo.It("should wait for sync to complete", func() {
			// Poll sync status until it changes from PENDING to IN_PROGRESS
			gomega.Eventually(func() string {
				var outputBuffer bytes.Buffer

				statusCmd := clicmd.RootCmd
				statusCmd.SetOut(&outputBuffer)
				statusCmd.SetArgs([]string{
					"sync",
					"status",
					syncID,
					"--server-addr",
					Peer2Addr,
				})

				err := statusCmd.Execute()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				output := outputBuffer.String()
				ginkgo.GinkgoWriter.Printf("Current sync status: %s", output)

				return output
			}, 120*time.Second, 5*time.Second).Should(gomega.ContainSubstring("IN_PROGRESS"))
		})

		ginkgo.It("should succeed to pull agent_v2.json from peer 2 after sync", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				agentCID,
				"--server-addr",
				Peer2Addr,
			})

			err := pullCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Compare the output with the expected JSON
			equal, err := compareJSONAgents(outputBuffer.Bytes(), expectedAgentV2JSON)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(equal).To(gomega.BeTrue())
		})

		// Delete sync from peer 2
		ginkgo.It("should delete sync from peer 2", func() {
			var outputBuffer bytes.Buffer

			deleteCmd := clicmd.RootCmd
			deleteCmd.SetOut(&outputBuffer)
			deleteCmd.SetArgs([]string{
				"sync",
				"delete",
				syncID,
				"--server-addr",
				Peer2Addr,
			})

			err := deleteCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		// Wait for sync to complete
		ginkgo.It("should wait for delete to complete", func() {
			// Poll sync status until it changes from DELETE_PENDING to DELETED
			gomega.Eventually(func() string {
				var outputBuffer bytes.Buffer

				statusCmd := clicmd.RootCmd
				statusCmd.SetOut(&outputBuffer)
				statusCmd.SetArgs([]string{
					"sync",
					"status",
					syncID,
					"--server-addr",
					Peer2Addr,
				})

				err := statusCmd.Execute()
				gomega.Expect(err).NotTo(gomega.HaveOccurred())

				output := outputBuffer.String()
				ginkgo.GinkgoWriter.Printf("Current sync status: %s", output)

				return output
			}, 120*time.Second, 5*time.Second).Should(gomega.ContainSubstring("DELETED"))
		})

		// Push agent_v3.json to peer 1
		ginkgo.It("should push agent_v3.json to peer 1", func() {
			var outputBuffer bytes.Buffer

			pushCmd := clicmd.RootCmd
			pushCmd.SetOut(&outputBuffer)
			pushCmd.SetArgs([]string{
				"push",
				"./testdata/agent_v3.json",
				"--server-addr",
				Peer1Addr,
			})

			err := pushCmd.Execute()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			agentCID = strings.TrimSpace(outputBuffer.String())
		})

		// Pull agent_v3.json from peer 2
		ginkgo.It("should fail to pull agent_v3.json from peer 2", func() {
			var outputBuffer bytes.Buffer

			pullCmd := clicmd.RootCmd
			pullCmd.SetOut(&outputBuffer)
			pullCmd.SetArgs([]string{
				"pull",
				agentCID,
				"--server-addr",
				Peer2Addr,
			})

			err := pullCmd.Execute()
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})
})
