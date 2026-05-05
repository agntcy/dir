// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"testing"

	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/tests/e2e/client/config"
	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var testEnv *env

type env struct {
	Config clientconfig.Config
	Client *client.Client
}

func TestClientE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	// Load configuration
	cfg, err := config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create client
	client, err := client.New(t.Context(), client.WithConfig(&cfg.Client.ClientOptions))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Set test environment
	testEnv = &env{
		Config: cfg.Client,
		Client: client,
	}

	// Run test suite
	ginkgo.RunSpecs(t, "Client Library E2E Test Suite")
}

var _ = ginkgo.BeforeSuite(func(ctx context.Context) {
	utils.WaitForGrpcServerReady(ctx, testEnv.Config.ClientOptions.ServerAddress)
})

var _ = ginkgo.AfterSuite(func() {
	err := testEnv.Client.Close()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
})
