// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"

	"github.com/agntcy/dir/client"
	policyconfig "github.com/agntcy/dir/tests/e2e/policy/config"
	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var testEnv *env

type env struct {
	Config policyconfig.Config
	Client *client.Client
}

func TestPolicyE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	cfg, err := config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	dirClient, err := client.New(t.Context(), client.WithConfig(&cfg.Policy.ClientOptions))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	testEnv = &env{
		Config: cfg.Policy,
		Client: dirClient,
	}

	ginkgo.RunSpecs(t, "Content Policy E2E Test Suite")
}

var _ = ginkgo.BeforeSuite(func(ctx context.Context) {
	utils.WaitForGrpcServerReady(
		ctx,
		testEnv.Config.ClientOptions.ServerAddress,
		testEnv.Config.ClientOptions.SpiffeSocketPath,
	)
})

var _ = ginkgo.AfterSuite(func() {
	err := testEnv.Client.Close()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
})
