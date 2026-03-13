// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func generatePublicKey() string {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	pub, ok := key.Public().(*rsa.PublicKey)
	gomega.Expect(ok).To(gomega.BeTrue())

	pubPkcs1 := x509.MarshalPKCS1PublicKey(pub)
	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubPkcs1,
	})

	return string(pubPem)
}

func generateRecord() *corev1.Record {
	record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV100JSON)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	name := record.GetData().GetFields()["name"].GetStringValue()
	record.Data.Fields["name"] = structpb.NewStringValue(name + "_" + uuid.NewString()[:8])

	return record
}

func generateReferrer() *corev1.RecordReferrer {
	publicKey := signv1.PublicKey{Key: generatePublicKey()}
	referrer, err := publicKey.MarshalReferrer()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return referrer
}

func pullReferrers(c *client.Client, ctx context.Context, recordRef *corev1.CID, referrerType string) []*corev1.RecordReferrer {
	ch, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef:    recordRef,
		ReferrerType: &referrerType,
	})

	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	referrers := []*corev1.RecordReferrer{}
	for response := range ch {
		referrers = append(referrers, response.GetReferrer())
	}

	return referrers
}

func expectError(err error, code codes.Code, msg string) {
	gomega.Expect(err).To(gomega.HaveOccurred())
	e, ok := status.FromError(err)
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(e.Code()).To(gomega.Equal(code))
	gomega.Expect(e.Message()).To(gomega.Equal(
		"failed to receive push referrer response: " +
			fmt.Sprintf("rpc error: code = %s desc = validation error: %s", code.String(), msg),
	))
}

var _ = ginkgo.Describe("Running e2e tests for referrers", func() {
	var (
		c       *client.Client
		ctx     context.Context
		record1 *corev1.CID
		record2 *corev1.CID
	)

	ginkgo.BeforeEach(func() {
		var err error

		ctx = context.Background()

		c, err = client.New(ctx, client.WithEnvConfig())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		record1, err = c.Push(ctx, generateRecord())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		record2, err = c.Push(ctx, generateRecord())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		c.Delete(ctx, record1) //nolint:errcheck
		c.Delete(ctx, record2) //nolint:errcheck
	})

	ginkgo.It("should successfully push basic referrer", func() {
		var err error

		referrer := generateReferrer()
		_, err = c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef: record1,
			Referrer:  referrer,
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		referrers := pullReferrers(c, ctx, record1, corev1.PublicKeyReferrerType)
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetRecordRef().GetCid()).To(gomega.Equal(record1.GetCid()))
		gomega.Expect(referrers[0].GetType()).To(gomega.Equal(corev1.PublicKeyReferrerType))
		gomega.Expect(referrers[0].GetAnnotations()).To(gomega.BeNil())
		gomega.Expect(referrers[0].GetCreatedAt()).To(gomega.Equal(""))
		gomega.Expect(referrers[0].GetData().AsMap()).To(gomega.Equal(referrer.GetData().AsMap()))
	})

	ginkgo.It("should successfully push full referrer", func() {
		var err error

		referrer := generateReferrer()
		referrer.CreatedAt = "2026-03-09T14:20:00Z"
		referrer.RecordRef = record1
		referrer.Annotations = map[string]string{"foo": "bar"}
		_, err = c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef: record1,
			Referrer:  referrer,
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		referrers := pullReferrers(c, ctx, record1, corev1.PublicKeyReferrerType)
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetRecordRef().GetCid()).To(gomega.Equal(record1.GetCid()))
		gomega.Expect(referrers[0].GetType()).To(gomega.Equal(corev1.PublicKeyReferrerType))
		gomega.Expect(referrers[0].GetAnnotations()).To(gomega.Equal(map[string]string{"foo": "bar"}))
		gomega.Expect(referrers[0].GetCreatedAt()).To(gomega.Equal("2026-03-09T14:20:00Z"))
		gomega.Expect(referrers[0].GetData().AsMap()).To(gomega.Equal(referrer.GetData().AsMap()))
	})

	ginkgo.It("should fail if record mismatch", func() {
		var err error

		referrer := generateReferrer()
		referrer.RecordRef = record2
		response, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef: record1,
			Referrer:  referrer,
		})

		// FIXME: make sure to return an error if there's an error
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(response.GetSuccess()).To(gomega.BeFalse())
		gomega.Expect(response.GetErrorMessage()).To(gomega.Equal(
			fmt.Sprintf("failed to push referrer for record %s: ", record1.GetCid()) +
				"rpc error: code = InvalidArgument desc = referrer's record CID must match record CID",
		))
	})

	ginkgo.It("should fail if empty referrer", func() {
		referrer := &corev1.RecordReferrer{}
		response, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef: record1,
			Referrer:  referrer,
		})

		msg := "referrer.type: value must be in list [agntcy.dir.sign.v1.PublicKey, agntcy.dir.sign.v1.Signature]"
		expectError(err, codes.InvalidArgument, msg)
		gomega.Expect(response.GetSuccess()).To(gomega.BeFalse())
		gomega.Expect(response.GetErrorMessage()).To(gomega.Equal(""))
	})

	ginkgo.It("should fail if invalid referrer type", func() {
		referrer := generateReferrer()
		referrer.Type = "foo"
		response, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef: record1,
			Referrer:  referrer,
		})

		msg := "referrer.type: value must be in list [agntcy.dir.sign.v1.PublicKey, agntcy.dir.sign.v1.Signature]"
		expectError(err, codes.InvalidArgument, msg)
		gomega.Expect(response.GetSuccess()).To(gomega.BeFalse())
		gomega.Expect(response.GetErrorMessage()).To(gomega.Equal(""))

		referrers := pullReferrers(c, ctx, record1, "foo")
		gomega.Expect(referrers).To(gomega.BeEmpty())
	})
})
