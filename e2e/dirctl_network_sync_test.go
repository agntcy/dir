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
)

var _ = ginkgo.Describe("Running dirctl end-to-end tests for sync commands", func() {
	ginkgo.BeforeEach(func() {
		if cfg.DeploymentMode != config.DeploymentModeLocal {
			ginkgo.Skip("Skipping test, not in local mode")
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
			})

			err := deleteCmd.Execute()
			// Command may fail due to network/auth issues, but argument parsing should work
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("required"))
			}
		})

		ginkgo.It("should not list the deleted sync", func() {
			var outputBuffer bytes.Buffer

			listCmd := clicmd.RootCmd
			listCmd.SetOut(&outputBuffer)
			listCmd.SetArgs([]string{
				"sync",
				"list",
			})

			err := listCmd.Execute()
			if err != nil {
				gomega.Expect(err.Error()).NotTo(gomega.ContainSubstring("required"))
			}

			gomega.Expect(outputBuffer.String()).NotTo(gomega.ContainSubstring(syncID))
		})
	})
})
