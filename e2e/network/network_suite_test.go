// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"testing"
	"time"

	"github.com/agntcy/dir/e2e/shared/config"
	"github.com/agntcy/dir/e2e/shared/utils"
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

// Warm-up phase - runs BEFORE all tests to prevent cold start failures.
// This ensures the Kubernetes environment is fully ready before any actual tests run.
var _ = ginkgo.BeforeSuite(func() {
	ginkgo.GinkgoWriter.Println("\n==============================================")
	ginkgo.GinkgoWriter.Println("  WARM-UP PHASE - Preparing environment...")
	ginkgo.GinkgoWriter.Println("==============================================")

	cli := utils.NewCLI()

	// Wait for pods to fully initialize
	ginkgo.GinkgoWriter.Println("⏳ Waiting for pods to initialize (30s)...")
	time.Sleep(30 * time.Second)

	// Verify all peers are reachable
	ginkgo.GinkgoWriter.Println("⏳ Verifying peer connectivity...")
	for i, peerAddr := range utils.PeerAddrs {
		ginkgo.GinkgoWriter.Printf("  Peer%d (%s)...", i+1, peerAddr)
		output := cli.Routing().Info().OnServer(peerAddr).ShouldSucceed()
		gomega.Expect(output).To(gomega.ContainSubstring("Routing"))
		ginkgo.GinkgoWriter.Println(" ✅")
	}

	// Wait for DHT mesh formation
	ginkgo.GinkgoWriter.Println("⏳ Waiting for DHT mesh formation (15s)...")
	time.Sleep(15 * time.Second)

	// Wait for GossipSub mesh formation
	ginkgo.GinkgoWriter.Println("⏳ Waiting for GossipSub mesh formation (10s)...")
	time.Sleep(10 * time.Second)

	// Final verification - ensure all routing systems functional
	ginkgo.GinkgoWriter.Println("⏳ Final routing system verification...")
	for i, peerAddr := range utils.PeerAddrs {
		utils.ResetCLIState()
		ginkgo.GinkgoWriter.Printf("  Peer%d routing...", i+1)
		cli.Routing().Info().OnServer(peerAddr).ShouldSucceed()
		ginkgo.GinkgoWriter.Println(" ✅")
	}

	// Final stabilization period
	ginkgo.GinkgoWriter.Println("⏳ Final stabilization (10s)...")
	time.Sleep(10 * time.Second)

	ginkgo.GinkgoWriter.Println("==============================================")
	ginkgo.GinkgoWriter.Println("  ✅ WARM-UP COMPLETE - Environment ready!")
	ginkgo.GinkgoWriter.Println("  Total warm-up time: ~75 seconds")
	ginkgo.GinkgoWriter.Println("==============================================\n")
})

// Final safety cleanup - runs after all network tests complete.
var _ = ginkgo.AfterSuite(func() {
	ginkgo.GinkgoWriter.Printf("Final network test suite cleanup (safety net)")
	CleanupAllNetworkTests()
})
