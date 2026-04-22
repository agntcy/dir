// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package network

import (
	"context"
	"testing"

	networkconfig "github.com/agntcy/dir/tests/e2e/network/config"
	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var testEnv *env

type env struct {
	Config networkconfig.Config
	Peer1  *utils.CLI
	Peer2  *utils.CLI
	Peer3  *utils.CLI
}

func (e *env) PeerAddresses() []string {
	return []string{e.Config.Peer1ServerAddress, e.Config.Peer2ServerAddress, e.Config.Peer3ServerAddress}
}

func (e *env) PeerCLIs() []*utils.CLI {
	return []*utils.CLI{e.Peer1, e.Peer2, e.Peer3}
}

func TestNetworkE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	// Load configuration
	cfg, err := config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Set test environment
	testEnv = &env{
		Config: cfg.Network,
		Peer1: utils.NewCLI(
			utils.WithPath(cfg.Network.CliPath),
			utils.WithArgs(cfg.Network.Peer1CliExtraArgs...),
		),
		Peer2: utils.NewCLI(
			utils.WithPath(cfg.Network.CliPath),
			utils.WithArgs(cfg.Network.Peer2CliExtraArgs...),
		),
		Peer3: utils.NewCLI(
			utils.WithPath(cfg.Network.CliPath),
			utils.WithArgs(cfg.Network.Peer3CliExtraArgs...),
		),
	}

	ginkgo.RunSpecs(t, "Network E2E Test Suite")
}

var _ = ginkgo.BeforeSuite(func(ctx context.Context) {
	for _, addr := range testEnv.PeerAddresses() {
		utils.WaitForGrpcServerReady(ctx, addr)
	}
})

var _ = ginkgo.AfterSuite(func() {
	ginkgo.GinkgoWriter.Printf("Final network test suite cleanup")
	CleanupAllNetworkTests(testEnv.PeerCLIs())
})
