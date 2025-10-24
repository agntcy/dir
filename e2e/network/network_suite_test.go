// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"testing"

	"github.com/agntcy/dir/e2e/shared/config"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var cfg *config.Config

// CID tracking variables are now in cleanup.go

func TestNetworkE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	var err error

	cfg, err = config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	if cfg.DeploymentMode != config.DeploymentModeNetwork {
		t.Skip("Skipping network tests - not in network mode")
	}

	ginkgo.RunSpecs(t, "Network E2E Test Suite")
}

// Final safety cleanup - runs after all network tests complete.
var _ = ginkgo.AfterSuite(func() {
	ginkgo.GinkgoWriter.Printf("Final network test suite cleanup (safety net)")
	CleanupAllNetworkTests()
})
