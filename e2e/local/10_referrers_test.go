// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Running e2e tests for referrers", func() {
	var (
		c         *client.Client
		ctx       context.Context
		err       error
		recordRef *corev1.RecordRef
	)

	publicKey := signv1.PublicKey{Key: testdata.PublicKey}

	ginkgo.BeforeEach(func() {
		ctx = context.Background()

		c, err = client.New(ctx, client.WithEnvConfig())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV070JSON)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		recordRef, err = c.Push(ctx, record)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		publicKeyReferrer, err := publicKey.MarshalReferrer()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		err = c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef: recordRef,
			Referrer:  publicKeyReferrer,
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.It("should successfully pull referrer", func() {
		referrerType := corev1.PublicKeyReferrerType
		ch, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
			RecordRef:    recordRef,
			ReferrerType: &referrerType,
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		responses := []*storev1.PullReferrerResponse{}
		for response := range ch {
			responses = append(responses, response)
		}

		gomega.Expect(responses).To(gomega.HaveLen(1))
		referrer := responses[0].GetReferrer()
		gomega.Expect(referrer).ToNot(gomega.BeNil())
		gomega.Expect(referrer.GetType()).To(gomega.Equal(corev1.PublicKeyReferrerType))
	})
})
