// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package integration_server

import (
	"testing"

	"github.com/agntcy/dir/tests/test_utils"
	"github.com/joho/godotenv"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const envFile = "server.env"

func TestIntegrationServer(t *testing.T) {
	err := godotenv.Load(envFile)
	if err != nil {
		t.Errorf("failed to load %s: %s", envFile, err)
	}

	gomega.RegisterFailHandler(ginkgo.Fail)
	test_utils.InitGofakeit()
	ginkgo.RunSpecs(t, "E2E Production (production-like) Test Suite")
}
