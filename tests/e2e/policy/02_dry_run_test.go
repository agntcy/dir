// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
)

// deployedPolicy is the policy the testenv enforces, as a candidate file.
const deployedPolicy = `
validators:
  - provider: cel
    op: ["evaluate"]
    config:
      name: has-description
      expressions:
        - 'record.description != ""'
`

// invertedPolicy rejects what the deployed policy passes, and passes what it
// rejects. A record pushed without a description has no description key at all,
// so the expression asks for the key first.
const invertedPolicy = `
validators:
  - provider: cel
    op: ["evaluate"]
    config:
      name: no-description
      expressions:
        - '!has(record.description) || record.description == ""'
`

// dryRunReport is what `dirctl daemon policy dry-run --output json` writes for
// one policy.
type dryRunReport struct {
	PolicyID     string `json:"policy_id"`
	Evaluated    int    `json:"evaluated"`
	Compliant    int    `json:"compliant"`
	NonCompliant int    `json:"non_compliant"`
	Failed       int    `json:"failed"`
	WouldExclude int    `json:"would_exclude"`
	Sample       []struct {
		CID    string `json:"cid"`
		Reason string `json:"reason"`
	} `json:"sample"`
}

func (r dryRunReport) excludes(ref *corev1.RecordRef) bool {
	for _, s := range r.Sample {
		if s.CID == ref.GetCid() {
			return true
		}
	}

	return false
}

// dryRun tries a candidate policy on the daemon's records with the command an
// operator would run, and returns its report.
func dryRun(candidate string) dryRunReport {
	ginkgo.GinkgoHelper()

	path := filepath.Join(ginkgo.GinkgoT().TempDir(), "candidate.yaml")
	gomega.Expect(os.WriteFile(path, []byte(candidate), 0o600)).To(gomega.Succeed())

	config, err := filepath.Abs(testEnv.Config.DaemonConfig)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	out := utils.NewCLI().Command("daemon").WithArgs(
		"policy", "dry-run",
		"--candidate", path,
		"--config", config,
		"--data-dir", testEnv.Config.DaemonDataDir,
		"--samples", "1000",
		"--output", "json",
	).ShouldSucceed()

	var parsed struct {
		Policies []dryRunReport `json:"policies"`
	}

	gomega.Expect(json.Unmarshal([]byte(out), &parsed)).To(gomega.Succeed(), out)
	gomega.Expect(parsed.Policies).To(gomega.HaveLen(1))

	return parsed.Policies[0]
}

var _ = ginkgo.Describe("Dry run of a candidate policy", ginkgo.Ordered, ginkgo.Serial, func() {
	var compliantRef, rejectedRef *corev1.RecordRef

	ginkgo.BeforeAll(func(ctx context.Context) {
		compliantRef = push(ctx, newRecord("dry-run-compliant", described))
		rejectedRef = push(ctx, newRecord("dry-run-rejected", ""))

		// Wait until the node serves the one the policy passes: both have had
		// their verdicts by then.
		gomega.Eventually(func() codes.Code { return pullCode(ctx, compliantRef) }).
			WithTimeout(settleTimeout).
			WithPolling(pollInterval).
			Should(gomega.Equal(codes.OK))
	})

	ginkgo.It("reports what a candidate policy would exclude", func() {
		report := dryRun(invertedPolicy)

		gomega.Expect(report.PolicyID).To(gomega.Equal("cel:no-description"))
		gomega.Expect(report.Evaluated).To(gomega.BeNumerically(">=", 2))
		gomega.Expect(report.WouldExclude).To(gomega.Equal(report.NonCompliant + report.Failed))
		gomega.Expect(report.Compliant + report.NonCompliant + report.Failed).To(gomega.Equal(report.Evaluated))

		// The record the node serves has a description, which this policy rejects.
		gomega.Expect(report.excludes(compliantRef)).To(gomega.BeTrue(), "the record with a description")
		gomega.Expect(report.excludes(rejectedRef)).To(gomega.BeFalse(), "the record without one passes: %+v (rejected=%s)", report, rejectedRef.GetCid())

		for _, s := range report.Sample {
			gomega.Expect(s.Reason).NotTo(gomega.BeEmpty(), "each record comes with the reason")
		}
	})

	// A dry run of the policy the node already enforces says what the node
	// withholds, which is how an operator can trust what it says of another.
	ginkgo.It("agrees with what the node withholds when the candidate is the deployed policy", func(ctx context.Context) {
		report := dryRun(deployedPolicy)

		gomega.Expect(report.excludes(rejectedRef)).To(gomega.BeTrue(), "the record the node withholds")
		gomega.Expect(report.excludes(compliantRef)).To(gomega.BeFalse(), "the record the node serves")

		_, err := testEnv.Client.Pull(ctx, rejectedRef)
		expectWithheld(err)
	})

	// The acceptance of the dry run: it changes no verdict and so nothing a read
	// returns.
	ginkgo.It("changes nothing a read returns", func(ctx context.Context) {
		dryRun(invertedPolicy)

		gomega.Expect(pullCode(ctx, compliantRef)).To(gomega.Equal(codes.OK), "still served, though the candidate would exclude it")

		_, err := testEnv.Client.Pull(ctx, rejectedRef)
		expectWithheld(err)

		name, ok := nameOf(ctx, compliantRef)
		gomega.Expect(ok).To(gomega.BeTrue())
		gomega.Expect(searchByName(ctx, name)).To(gomega.ContainElement(compliantRef.GetCid()), "still found by search")
	})
})

// nameOf returns the name of a served record.
func nameOf(ctx context.Context, ref *corev1.RecordRef) (string, bool) {
	record, err := testEnv.Client.Pull(ctx, ref)
	if err != nil {
		return "", false
	}

	name, ok := record.GetData().AsMap()["name"].(string)

	return name, ok
}
