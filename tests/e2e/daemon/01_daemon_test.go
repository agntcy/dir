// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var (
	// Sample record for runtim discovery.
	sampleRuntimeRecord    = testdata.ExpectedRecordV100JSON
	sampleRuntimeRecordCID = "baeareiabbog2umgduqhlcb64fzt6adn34kblzvru3fdzkl75hjhwt6h3da"
)

// runtimeWorkloadService is the compose service holding the labelled container
// the runtime discovery specs look for, and runtimeWorkloadProfile is the
// profile keeping the testenv from starting it, so this suite controls when it
// appears.
const (
	runtimeWorkloadService = "service-a2a"
	runtimeWorkloadProfile = "runtime-discovery"
)

func cosignAvailable() bool {
	_, err := exec.LookPath("cosign")

	return err == nil
}

// composeWaitTimeout bounds the compose call below. It has to cover an image
// pull on a cold runner, which moved into the suite when the service went
// behind a profile. Without it a stalled pull is indistinguishable from a hung
// suite until the two-hour go test timeout fires.
const composeWaitTimeout = 90 * time.Second

// startRuntimeWorkload brings up the labelled container.
//
// Ordering matters and is the substance of issue #1780. The daemon's OASF
// resolver gives each workload a bounded retry budget, roughly 75s to 255s
// depending on whether attempts fail fast or time out, and persists
// `{"error": ...}` when the budget runs out. Nothing re-queues a workload
// afterwards, so a workload discovered before its record exists can never
// recover and the spec below is guaranteed to fail rather than merely be slow.
//
// Starting the container after the record has been pushed inverts that: the
// docker start event is what queues the workload, and the record is already in
// the store when the first resolve attempt runs.
func startRuntimeWorkload(ctx context.Context, composeFile string) {
	ginkgo.GinkgoHelper()

	gomega.Expect(composeFile).NotTo(gomega.BeEmpty(),
		"runtime_workload_compose_file must be set when runtime discovery tests are enabled")

	cmdCtx, cancel := context.WithTimeout(ctx, composeWaitTimeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "docker", "compose",
		"-f", composeFile,
		"--profile", runtimeWorkloadProfile,
		"up", "--wait", "--wait-timeout", strconv.Itoa(int(composeWaitTimeout.Seconds())),
		"-d", runtimeWorkloadService,
	)

	out, err := cmd.CombinedOutput()
	ginkgo.GinkgoWriter.Printf("docker compose up %s:\n%s\n", runtimeWorkloadService, string(out))

	gomega.Expect(err).NotTo(gomega.HaveOccurred(),
		"failed to start %s: %s", runtimeWorkloadService, string(out))
}

var _ = ginkgo.Describe("Daemon e2e", ginkgo.Ordered, ginkgo.Serial, func() {
	var (
		recordRef     *corev1.RecordRef
		canonicalData []byte
	)

	ginkgo.It("should push a record to the store", func(ctx context.Context) {
		record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		canonicalData, err = record.Marshal()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		recordRef, err = testEnv.Client.Push(ctx, record)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(recordRef).NotTo(gomega.BeNil())
		gomega.Expect(recordRef.GetCid()).NotTo(gomega.BeEmpty())

		utils.ValidateCIDAgainstData(recordRef.GetCid(), canonicalData)
	})

	ginkgo.It("should pull the pushed record back", func(ctx context.Context) {
		gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

		pulled, err := testEnv.Client.Pull(ctx, recordRef)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pulledCanonical, err := pulled.Marshal()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		equal, err := utils.CompareOASFRecords(canonicalData, pulledCanonical)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(equal).To(gomega.BeTrue(), "pushed and pulled records should be identical")
	})

	ginkgo.Context("signature workflow", ginkgo.Ordered, func() {
		var (
			cosignDir      string
			cosignPassword = "testpassword"
			keyPath        string
			pubKeyPath     string
		)

		ginkgo.BeforeAll(func() {
			if !cosignAvailable() {
				ginkgo.Skip("cosign binary not found, skipping signature tests")
			}

			gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

			cosignDir = ginkgo.GinkgoT().TempDir()
			keyPath = filepath.Join(cosignDir, "cosign.key")
			pubKeyPath = filepath.Join(cosignDir, "cosign.pub")

			utils.GenerateCosignKeyPair(cosignDir, cosignPassword)
			gomega.Expect(keyPath).To(gomega.BeAnExistingFile())
			gomega.Expect(pubKeyPath).To(gomega.BeAnExistingFile())

			err := os.Setenv("COSIGN_PASSWORD", cosignPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			os.Unsetenv("COSIGN_PASSWORD")
		})

		ginkgo.It("should sign the record with a key pair", func(ctx context.Context) {
			resp, err := testEnv.Client.Sign(ctx, &signv1.SignRequest{
				RecordRef: recordRef,
				Provider: &signv1.SignRequestProvider{
					Request: &signv1.SignRequestProvider_Key{
						Key: &signv1.SignWithKey{
							PrivateKey: keyPath,
							Password:   []byte(cosignPassword),
						},
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSignature()).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSignature().GetSignature()).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should verify the signature with the public key", func(ctx context.Context) {
			resp, err := testEnv.Client.Verify(ctx, &signv1.VerifyRequest{
				RecordRef: recordRef,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSuccess()).To(gomega.BeTrue(), "signature verification should succeed")
			gomega.Expect(resp.GetSigners()).NotTo(gomega.BeEmpty())
		})
	})

	ginkgo.Context("runtime workflow", ginkgo.Ordered, func() {
		// Push the record first, then start the workload that references it.
		// Both halves matter: see startRuntimeWorkload for why the reverse
		// order cannot recover.
		ginkgo.BeforeAll(func(ctx context.Context) {
			if !testEnv.Config.RunRuntimeDiscoveryTests {
				ginkgo.Skip("skipping runtime tests")
			}

			record, err := corev1.UnmarshalRecord(sampleRuntimeRecord)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordRef, err := testEnv.Client.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(recordRef).NotTo(gomega.BeNil())
			gomega.Expect(recordRef.GetCid()).To(gomega.Equal(sampleRuntimeRecordCID))

			startRuntimeWorkload(ctx, testEnv.Config.RuntimeWorkloadComposeFile)
		}, ginkgo.NodeTimeout(2*composeWaitTimeout))

		// Registration is asserted separately from resolution so the two
		// failure modes stay distinguishable. Registration depends only on the
		// daemon receiving the docker start event; if the watcher was not
		// listening yet the event is lost, because dir-runtime subscribes to
		// the docker event stream without a `since` and only reconciles at
		// boot. That failure should read as "the workload never appeared",
		// not as a three-minute timeout that looks like a resolver problem.
		ginkgo.It("runtime should register the docker workload", func(ctx context.Context) {
			gomega.Eventually(func(g gomega.Gomega) {
				discoverResp, err := testEnv.Client.ListWorkloads(ctx, nil)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(discoverResp).To(gomega.HaveLen(1))

				workload := discoverResp[0]
				g.Expect(workload.GetType()).To(gomega.Equal("container"))
				g.Expect(workload.GetRuntime()).To(gomega.Equal("docker"))
				g.Expect(workload.GetLabels()).To(gomega.HaveKeyWithValue("test.org.agntcy/agent-type", "a2a"))
				g.Expect(workload.GetLabels()).To(gomega.HaveKeyWithValue("test.org.agntcy/agent-record", sampleRuntimeRecordCID))
			}).
				WithPolling(2 * time.Second).
				WithTimeout(1 * time.Minute).
				Should(gomega.Succeed())
		})

		// The resolver's own budget is minutes long, so this gets the longer
		// timeout. It can only pass if the record was already in the store
		// when the workload was queued, which the BeforeAll ordering ensures.
		ginkgo.It("runtime should resolve the workload's OASF record", func(ctx context.Context) {
			gomega.Eventually(func(g gomega.Gomega) {
				discoverResp, err := testEnv.Client.ListWorkloads(ctx, nil)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(discoverResp).To(gomega.HaveLen(1))

				workload := discoverResp[0]
				g.Expect(workload.GetServices()).NotTo(gomega.BeNil())

				// Validate OASF service discovery data. AsMap returns an empty
				// non-nil map for an unset Oasf, so this starts empty rather
				// than nil while the a2a resolver has run but the OASF one
				// has not; the key assertions below are what gate on it.
				oasfService := workload.GetServices().GetOasf().AsMap()

				// A resolver that exhausted its retries persists an "error"
				// key here instead of the record. Surface it rather than
				// letting it fail as a missing "cid".
				g.Expect(oasfService).NotTo(gomega.HaveKey("error"),
					"resolver gave up before the record was reachable: %v", oasfService["error"])

				g.Expect(oasfService).To(gomega.HaveKeyWithValue("cid", sampleRuntimeRecordCID))
				g.Expect(oasfService).To(gomega.HaveKey("record"))

				// Validate discovered OASF record matches the original record
				oasfRecord, err := json.Marshal(oasfService["record"])
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(oasfRecord).To(gomega.MatchJSON(sampleRuntimeRecord))
			}).
				WithPolling(5 * time.Second).
				WithTimeout(3 * time.Minute).
				Should(gomega.Succeed())
		})
	})
})
