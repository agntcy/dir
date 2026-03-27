// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"testing"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/config"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

const (
	defaultServerAddress = "localhost:8888"
	readyPollInterval    = 2 * time.Second
	readyTimeout         = 2 * time.Minute
)

var cfg *config.Config

func TestLocalE2E(t *testing.T) {
	format.TruncatedDiff = false

	gomega.RegisterFailHandler(ginkgo.Fail)

	var err error

	cfg, err = config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	if cfg.DeploymentMode != config.DeploymentModeLocal {
		t.Skip("Skipping local tests - not in local mode")
	}

	ginkgo.RunSpecs(t, "Local E2E Test Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	addr := os.Getenv("DIRECTORY_CLIENT_SERVER_ADDRESS")
	if addr == "" {
		addr = defaultServerAddress
	}

	ginkgo.GinkgoWriter.Printf("Waiting for Directory apiserver at %s...\n", addr)
	gomega.Eventually(utils.IsGrpcServerReady).
		WithArguments(addr).
		WithPolling(readyPollInterval).
		WithTimeout(readyTimeout).
		Should(gomega.Succeed())
	ginkgo.GinkgoWriter.Printf("Directory apiserver is ready at %s\n", addr)
})
