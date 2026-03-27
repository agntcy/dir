// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

const cosignTestPassword = "testpassword"

func cosignAvailable() bool {
	_, err := exec.LookPath("cosign")

	return err == nil
}

var _ = ginkgo.Describe("Daemon e2e", ginkgo.Ordered, ginkgo.Serial, func() {
	var (
		c   *client.Client
		ctx context.Context

		recordRef     *corev1.RecordRef
		canonicalData []byte
	)

	ginkgo.BeforeAll(func() {
		ctx = context.Background()

		var err error

		c, err = client.New(ctx, client.WithEnvConfig())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterAll(func() {
		if c != nil {
			_ = c.Close()
		}
	})

	ginkgo.It("should push a record to the store", func() {
		record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		canonicalData, err = record.Marshal()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		recordRef, err = c.Push(ctx, record)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(recordRef).NotTo(gomega.BeNil())
		gomega.Expect(recordRef.GetCid()).NotTo(gomega.BeEmpty())

		utils.ValidateCIDAgainstData(recordRef.GetCid(), canonicalData)
	})

	ginkgo.It("should pull the pushed record back", func() {
		gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

		pulled, err := c.Pull(ctx, recordRef)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		pulledCanonical, err := pulled.Marshal()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		equal, err := utils.CompareOASFRecords(canonicalData, pulledCanonical)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(equal).To(gomega.BeTrue(), "pushed and pulled records should be identical")
	})

	ginkgo.Context("signature workflow", ginkgo.Ordered, func() {
		var (
			cosignDir  string
			keyPath    string
			pubKeyPath string
		)

		ginkgo.BeforeAll(func() {
			if !cosignAvailable() {
				ginkgo.Skip("cosign binary not found, skipping signature tests")
			}

			gomega.Expect(recordRef).NotTo(gomega.BeNil(), "push must succeed first")

			cosignDir = ginkgo.GinkgoT().TempDir()
			keyPath = filepath.Join(cosignDir, "cosign.key")
			pubKeyPath = filepath.Join(cosignDir, "cosign.pub")

			utils.GenerateCosignKeyPair(cosignDir)
			gomega.Expect(keyPath).To(gomega.BeAnExistingFile())
			gomega.Expect(pubKeyPath).To(gomega.BeAnExistingFile())

			err := os.Setenv("COSIGN_PASSWORD", cosignTestPassword)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		})

		ginkgo.AfterAll(func() {
			os.Unsetenv("COSIGN_PASSWORD")
		})

		ginkgo.It("should sign the record with a key pair", func() {
			resp, err := c.Sign(ctx, &signv1.SignRequest{
				RecordRef: recordRef,
				Provider: &signv1.SignRequestProvider{
					Request: &signv1.SignRequestProvider_Key{
						Key: &signv1.SignWithKey{
							PrivateKey: keyPath,
							Password:   []byte(cosignTestPassword),
						},
					},
				},
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSignature()).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSignature().GetSignature()).NotTo(gomega.BeEmpty())
		})

		ginkgo.It("should verify the signature with the public key", func() {
			resp, err := c.Verify(ctx, &signv1.VerifyRequest{
				RecordRef: recordRef,
			})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp).NotTo(gomega.BeNil())
			gomega.Expect(resp.GetSuccess()).To(gomega.BeTrue(), "signature verification should succeed")
			gomega.Expect(resp.GetSigners()).NotTo(gomega.BeEmpty())
		})
	})
})
