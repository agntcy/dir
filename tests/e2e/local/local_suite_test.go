// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"testing"

	localconfig "github.com/agntcy/dir/tests/e2e/local/config"
	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var testEnv *env

type env struct {
	Config localconfig.Config
	CLI    *utils.CLI
}

func TestLocalE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	// Load configuration
	cfg, err := config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create CLI
	cli := utils.NewCLI(
		utils.WithPath(cfg.Local.CliPath),
		utils.WithArgs(cfg.Local.CliExtraArgs...),
	)

	// Set test environment
	testEnv = &env{
		Config: cfg.Local,
		CLI:    cli,
	}

	ginkgo.RunSpecs(t, "Local E2E Test Suite")
}

var _ = ginkgo.BeforeSuite(func(ctx context.Context) {
	utils.WaitForGrpcServerReady(ctx, testEnv.Config.ServerAddress)
})
