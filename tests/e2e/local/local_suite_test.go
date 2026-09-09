// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"net/http"
	"testing"
	"time"

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

// extractorReadyTimeout bounds the wait for the gateway's extractor to answer.
// The remote backend fetches the taxonomy from the OASF schema endpoint on every
// start and cannot serve until it finishes, which takes roughly a minute. The
// local backend answers as soon as the daemon is up, so this only ever costs
// time under kind.
//
// Waiting once here rather than per spec keeps a cold extractor pod from
// failing whichever extractor-backed spec happens to run first.
const extractorReadyTimeout = 3 * time.Minute

var _ = ginkgo.BeforeSuite(func(ctx context.Context) {
	utils.WaitForGrpcServerReady(ctx, testEnv.Config.ServerAddress, "")

	waitForExtractorReady(ctx)
})

// waitForExtractorReady blocks until POST /v1/extract answers, so the
// extractor-backed specs fail on behaviour rather than on a backend that had not
// finished starting. It is a no-op in environments with no gateway or no
// declared extractor.
func waitForExtractorReady(ctx context.Context) {
	if testEnv.Config.GatewayAddress == "" || testEnv.Config.ExtractorMode == "" {
		return
	}

	ginkgo.GinkgoWriter.Printf("waiting for the %s OASF extractor behind %s\n",
		testEnv.Config.ExtractorMode, testEnv.Config.GatewayAddress)

	gomega.Eventually(func(g gomega.Gomega) {
		status, body, err := tryGatewayPostJSON(ctx, "/v1/extract", map[string]string{
			"text": "an agent that reviews source code",
		})
		g.Expect(err).NotTo(gomega.HaveOccurred(), "gateway not reachable yet")
		g.Expect(status).To(gomega.Equal(http.StatusOK), "extractor not ready yet: %s", body)
	}).WithContext(ctx).
		WithTimeout(extractorReadyTimeout).
		WithPolling(5*time.Second).
		Should(gomega.Succeed(), "the gateway's %s extractor never became ready", testEnv.Config.ExtractorMode)
}
