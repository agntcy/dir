// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"testing"

	"github.com/agntcy/dir/e2e/config"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var cfg *config.Config

func TestEndToEnd(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	var err error
	cfg, err = config.LoadConfig()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	ginkgo.RunSpecs(t, "Run end-to-end tests")
}
