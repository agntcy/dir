// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

func cosignAvailable() bool {
	_, err := exec.LookPath("cosign")

	return err == nil
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

	ginkgo.Context("runtime workflow", func() {
		ginkgo.BeforeAll(func() {
			if !testEnv.Config.RunRuntimeDiscoveryTests {
				ginkgo.Skip("skipping runtime tests")
			}
		})

		ginkgo.It("runtime should discover docker workloads", func(ctx context.Context) {
			// Push sample record to ensure there's something to discover and pull back for verification
			record, err := corev1.UnmarshalRecord(sampleRuntimeRecord)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordRef, err := testEnv.Client.Push(ctx, record)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(recordRef).NotTo(gomega.BeNil())
			gomega.Expect(recordRef.GetCid()).To(gomega.Equal(sampleRuntimeRecordCID))

			// Use eventually to allow some time for the runtime to discover the workload and its service details.
			// The test will poll the ListWorkloads API until it finds the expected workload with the correct OASF data, or until it times out.
			gomega.Eventually(func(g gomega.Gomega) {
				// Discover runtimes workloads
				discoverResp, err := testEnv.Client.ListWorkloads(ctx, nil)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(discoverResp).NotTo(gomega.BeNil())
				g.Expect(discoverResp).To(gomega.HaveLen(1))

				// Validate workload
				workload := discoverResp[0]
				g.Expect(workload.GetType()).To(gomega.Equal("container"))
				g.Expect(workload.GetRuntime()).To(gomega.Equal("docker"))
				g.Expect(workload.GetLabels()).To(gomega.HaveKeyWithValue("test.org.agntcy/agent-type", "a2a"))
				g.Expect(workload.GetLabels()).To(gomega.HaveKeyWithValue("test.org.agntcy/agent-record", sampleRuntimeRecordCID))
				g.Expect(workload.GetServices()).To(gomega.Not(gomega.BeNil()))

				// Validate OASF service discovery data
				oasfService := workload.GetServices().GetOasf().AsMap()
				g.Expect(oasfService).NotTo(gomega.BeNil())
				g.Expect(oasfService).To(gomega.HaveKeyWithValue("cid", sampleRuntimeRecordCID))
				g.Expect(oasfService).To(gomega.HaveKey("record"))

				// Validate discovered OASF record matches the original record
				oasfRecord, err := json.Marshal(oasfService["record"])
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(oasfRecord).To(gomega.MatchJSON(sampleRuntimeRecord))
			}).
				WithPolling(10 * time.Second).
				WithTimeout(2 * time.Minute).
				Should(gomega.Succeed())
		})
	})
})
