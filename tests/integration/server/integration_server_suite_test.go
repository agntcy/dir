// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	"github.com/agntcy/dir/tests/test_utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestIntegrationServer(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	test_utils.InitGofakeit()
	ginkgo.RunSpecs(t, "Integration Test Suite")
}
