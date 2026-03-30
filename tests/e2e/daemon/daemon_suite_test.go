// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const (
	defaultServerAddress = "localhost:8888"
	readyPollInterval    = 1 * time.Second
	readyTimeout         = 30 * time.Second
)

func TestDaemonE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Daemon E2E Test Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	addr := os.Getenv("DIRECTORY_CLIENT_SERVER_ADDRESS")
	if addr == "" {
		addr = defaultServerAddress
	}

	ginkgo.GinkgoWriter.Printf("Waiting for daemon at %s...\n", addr)
	gomega.Expect(waitForServerReady(addr)).To(gomega.Succeed())
	ginkgo.GinkgoWriter.Printf("Daemon is ready at %s\n", addr)
})

func waitForServerReady(addr string) error {
	deadline := time.Now().Add(readyTimeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second) //nolint:gosec,noctx,mnd
		if err == nil {
			conn.Close()

			return nil
		}

		time.Sleep(readyPollInterval)
	}

	return fmt.Errorf("daemon did not become ready at %s within %v", addr, readyTimeout)
}
