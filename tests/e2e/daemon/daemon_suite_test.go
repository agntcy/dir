// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"testing"

	"github.com/agntcy/dir/client"
	daemonconfig "github.com/agntcy/dir/tests/e2e/daemon/config"
	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var testEnv *env

type env struct {
	Config daemonconfig.Config
	Client *client.Client
}

func TestDaemonE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	// Load configuration
	cfg, err := config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create client
	client, err := client.New(t.Context(), client.WithConfig(&cfg.Daemon.ClientOptions))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Set test environment
	testEnv = &env{
		Config: cfg.Daemon,
		Client: client,
	}

	ginkgo.RunSpecs(t, "Daemon E2E Test Suite")
}

var _ = ginkgo.BeforeSuite(func(ctx context.Context) {
	utils.WaitForGrpcServerReady(ctx, testEnv.Config.ClientOptions.ServerAddress)
})

var _ = ginkgo.AfterSuite(func() {
	err := testEnv.Client.Close()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
})
