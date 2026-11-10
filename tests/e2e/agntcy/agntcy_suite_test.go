// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agntcy_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

type suiteConfig struct {
	dirctl, verifier, node string
	artifacts              string
	portBase               int
}

var config suiteConfig

func TestAGNTCYE2E(t *testing.T) {
	if os.Getenv("AGNTCY_E2E") != "1" {
		t.Skip("opt-in live suite: set AGNTCY_E2E=1 and the binary paths documented in README.md")
	}

	gomega.RegisterFailHandler(ginkgo.Fail)

	config = suiteConfig{
		dirctl:   binaryPath(t, "AGNTCY_E2E_DIRCTL"),
		verifier: binaryPath(t, "AGNTCY_E2E_VERIFIER"),
		node:     binaryPath(t, "AGNTCY_E2E_NODE"),
		portBase: 19300,
	}

	if port := os.Getenv("AGNTCY_E2E_PORT_BASE"); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1024 || value > 65530 {
			t.Fatalf("invalid AGNTCY_E2E_PORT_BASE %q: expected 1024–65530", port)
		}

		config.portBase = value
	}

	config.artifacts = os.Getenv("AGNTCY_E2E_ARTIFACTS")
	if config.artifacts == "" {
		t.Fatal("AGNTCY_E2E_ARTIFACTS must name a directory for container logs and test evidence")
	}

	var err error

	config.artifacts, err = filepath.Abs(config.artifacts)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(config.artifacts, 0o700); err != nil {
		t.Fatal(err)
	}

	ginkgo.RunSpecs(t, "AGNTCY Identity Live E2E Suite")
}

func binaryPath(t *testing.T, variable string) string {
	t.Helper()

	value := os.Getenv(variable)
	if value == "" {
		t.Fatalf("%s must name a Linux binary matching Docker's architecture", variable)
	}

	path, err := filepath.Abs(value)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path) //nolint:gosec // The operator explicitly selects a local fixture binary.
	if err != nil {
		t.Fatal(err)
	}

	if !info.Mode().IsRegular() {
		t.Fatalf("%s is not a regular file: %s", variable, path)
	}

	return path
}

var _ = ginkgo.Describe("AGNTCY identity with real services", ginkgo.Ordered, ginkgo.Serial, ginkgo.Label("agntcy", "live"), func() {
	var (
		fixture *liveFixture
		good    string
	)

	ginkgo.BeforeAll(func(ctx ginkgo.SpecContext) {
		fixture = newLiveFixture(config)
		ginkgo.DeferCleanup(fixture.cleanup)
		fixture.start(ctx)
	}, ginkgo.NodeTimeout(5*time.Minute))

	ginkgo.It("enrolls an identity with real OIDC proof and resolves it through Node", func(ctx ginkgo.SpecContext) {
		fixture.checkIdentity(ctx)
		ginkgo.AddReportEntry("agent_subject", fixture.subject)
	})

	ginkgo.It("excludes an unclaimed record, then verifies its native claim and matching badge", func(ctx ginkgo.SpecContext) {
		good = fixture.push(ctx, "valid", 10*time.Minute)
		gomega.Expect(fixture.search(ctx)).NotTo(gomega.ContainSubstring(good))
		fixture.claim(ctx, good, "agent.key")
		fixture.waitStatus(ctx, good, "verified")
		gomega.Expect(fixture.search(ctx)).To(gomega.ContainSubstring(good))
		ginkgo.AddReportEntry("valid_cid", good)
	})

	ginkgo.It("stores the native OCI claim with exact record manifest, CID and subject binding", func(ctx ginkgo.SpecContext) {
		fixture.checkStorage(ctx, good)
	})

	ginkgo.It("rejects reuse of another record's badge under the same Agent ID", func(ctx ginkgo.SpecContext) {
		cid := fixture.push(ctx, "no-matching-badge", 0)
		fixture.claim(ctx, cid, "agent.key")
		fixture.waitStatus(ctx, cid, "failed")
		gomega.Expect(fixture.search(ctx)).NotTo(gomega.ContainSubstring(cid))
		ginkgo.AddReportEntry("badge_mismatch_cid", cid)
	})

	ginkgo.It("rejects an unrelated claim signer even with a valid badge", func(ctx ginkgo.SpecContext) {
		fixture.privateKey("wrong.key")
		cid := fixture.push(ctx, "wrong-key", 10*time.Minute)
		fixture.claim(ctx, cid, "wrong.key")
		fixture.waitStatus(ctx, cid, "failed")
		gomega.Expect(fixture.search(ctx)).NotTo(gomega.ContainSubstring(cid))
		ginkgo.AddReportEntry("wrong_key_cid", cid)
	})

	ginkgo.It("excludes an expired badge from verification and verified search", func(ctx ginkgo.SpecContext) {
		cid := fixture.push(ctx, "expired-badge", -time.Second)
		fixture.claim(ctx, cid, "agent.key")
		fixture.waitStatus(ctx, cid, "failed")
		gomega.Expect(fixture.search(ctx)).NotTo(gomega.ContainSubstring(cid))
		ginkgo.AddReportEntry("expired_badge_cid", cid)
	})

	ginkgo.It("honors the signed deadline during verifier outage and recovers after restart", func(ctx ginkgo.SpecContext) {
		fixture.waitStatus(ctx, good, "verified")
		fixture.docker(ctx, "stop", fixture.prefix+"-verifier")
		fixture.waitStatus(ctx, good, "failed")
		gomega.Expect(fixture.search(ctx)).NotTo(gomega.ContainSubstring(good))
		fixture.docker(ctx, "start", fixture.prefix+"-verifier")
		fixture.waitStatus(ctx, good, "verified")
		gomega.Expect(fixture.search(ctx)).To(gomega.ContainSubstring(good))
	})
})
