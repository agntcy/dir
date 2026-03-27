// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"os"
	"testing"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	defaultServerAddress = "localhost:8888"
	readyPollInterval    = 2 * time.Second
	readyTimeout         = 2 * time.Minute
)

func TestMCPE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "MCP E2E Test Suite")
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
