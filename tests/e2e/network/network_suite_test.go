// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"testing"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	readyPollInterval = 2 * time.Second
	readyTimeout      = 2 * time.Minute
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

var _ = ginkgo.BeforeSuite(func() {
	for _, addr := range utils.PeerAddrs {
		ginkgo.GinkgoWriter.Printf("Waiting for Directory apiserver at %s...\n", addr)
		gomega.Eventually(utils.IsGrpcServerReady).
			WithArguments(addr).
			WithPolling(readyPollInterval).
			WithTimeout(readyTimeout).
			Should(gomega.Succeed())
		ginkgo.GinkgoWriter.Printf("Directory apiserver is ready at %s\n", addr)
	}
})

// Final safety cleanup - runs after all network tests complete.
var _ = ginkgo.AfterSuite(func() {
	ginkgo.GinkgoWriter.Printf("Final network test suite cleanup (safety net)")
	CleanupAllNetworkTests()
})
