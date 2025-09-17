// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"testing"

	"github.com/agntcy/dir/e2e/shared/config"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var cfg *config.Config

func TestClientE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	var err error
	cfg, err = config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	if cfg.DeploymentMode != config.DeploymentModeLocal {
		t.Skip("Skipping client tests - not in local mode")
	}

	ginkgo.RunSpecs(t, "Client Library E2E Test Suite")
}
