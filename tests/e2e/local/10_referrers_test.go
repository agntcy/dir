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
	"io"
	"strings"

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

func pullReferrers(c *client.Client, ctx context.Context, recordRef *corev1.RecordRef) []*corev1.RecordReferrer {
	referrerType := corev1.PublicKeyReferrerType
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
			fmt.Sprintf("rpc error: code = %s desc = %s", code.String(), msg),
	))
}

var _ = ginkgo.Describe("Running e2e tests for referrers", func() {
	var (
		c       *client.Client
		ctx     context.Context
		record1 *corev1.RecordRef
		record2 *corev1.RecordRef
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
		referrer := generateReferrer()
		response, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef:   record1,
			Type:        referrer.GetType(),
			Annotations: referrer.GetAnnotations(),
			CreatedAt:   referrer.GetCreatedAt(),
			Data:        referrer.GetData(),
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(response.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response.GetErrorMessage()).To(gomega.BeEmpty())
		gomega.Expect(response.GetReferrerRef().GetCid()).NotTo(gomega.BeNil())
		gomega.Expect(response.GetReferrerRef().GetCid()).NotTo(gomega.BeEmpty())

		referrers := pullReferrers(c, ctx, record1)
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetRecordRef().GetCid()).To(gomega.Equal(record1.GetCid()))
		gomega.Expect(referrers[0].GetType()).To(gomega.Equal(corev1.PublicKeyReferrerType))
		gomega.Expect(referrers[0].GetAnnotations()).To(gomega.BeNil())
		gomega.Expect(referrers[0].GetCreatedAt()).To(gomega.Equal(""))
		gomega.Expect(referrers[0].GetData().AsMap()).To(gomega.Equal(referrer.GetData().AsMap()))
	})

	ginkgo.It("should successfully push full referrer", func() {
		referrer := generateReferrer()
		referrer.CreatedAt = "2026-03-09T14:20:00Z"
		referrer.RecordRef = record1
		referrer.Annotations = map[string]string{"foo": "bar"}
		response, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef:   record1,
			Type:        referrer.GetType(),
			Annotations: referrer.GetAnnotations(),
			CreatedAt:   referrer.GetCreatedAt(),
			Data:        referrer.GetData(),
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(response.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response.GetErrorMessage()).To(gomega.BeEmpty())
		gomega.Expect(response.GetReferrerRef().GetCid()).NotTo(gomega.BeNil())
		gomega.Expect(response.GetReferrerRef().GetCid()).NotTo(gomega.BeEmpty())

		// Validate CID
		cid := response.GetReferrerRef().GetCid()
		gomega.Expect(cid).NotTo(gomega.BeEmpty())

		referrerBytes, err := referrer.Marshal()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		referrerDigest, err := corev1.CalculateDigest(referrerBytes)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		expectedCID, err := corev1.ConvertDigestToCID(referrerDigest)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(cid).To(gomega.Equal(expectedCID))

		referrers := pullReferrers(c, ctx, record1)
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetRecordRef().GetCid()).To(gomega.Equal(record1.GetCid()))
		gomega.Expect(referrers[0].GetType()).To(gomega.Equal(corev1.PublicKeyReferrerType))
		gomega.Expect(referrers[0].GetAnnotations()).To(gomega.Equal(map[string]string{"foo": "bar"}))
		gomega.Expect(referrers[0].GetCreatedAt()).To(gomega.Equal("2026-03-09T14:20:00Z"))
		gomega.Expect(referrers[0].GetData().AsMap()).To(gomega.Equal(referrer.GetData().AsMap()))
	})

	ginkgo.It("should pass if referrer exists", func() {
		referrer := generateReferrer()

		response1, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef:   record1,
			Type:        referrer.GetType(),
			Annotations: referrer.GetAnnotations(),
			CreatedAt:   referrer.GetCreatedAt(),
			Data:        referrer.GetData(),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid1 := response1.GetReferrerRef().GetCid()
		gomega.Expect(cid1).ToNot(gomega.BeEmpty())

		response2, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef:   record1,
			Type:        referrer.GetType(),
			Annotations: referrer.GetAnnotations(),
			CreatedAt:   referrer.GetCreatedAt(),
			Data:        referrer.GetData(),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid2 := response2.GetReferrerRef().GetCid()
		gomega.Expect(cid2).To(gomega.Equal(cid1))

		referrers := pullReferrers(c, ctx, record1)
		gomega.Expect(referrers).To(gomega.HaveLen(1))
	})

	ginkgo.It("should pass if same referrer different records", func() {
		referrer := generateReferrer()

		response1, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef:   record1,
			Type:        referrer.GetType(),
			Annotations: referrer.GetAnnotations(),
			CreatedAt:   referrer.GetCreatedAt(),
			Data:        referrer.GetData(),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid1 := response1.GetReferrerRef().GetCid()
		gomega.Expect(cid1).ToNot(gomega.BeEmpty())

		response2, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef:   record2,
			Type:        referrer.GetType(),
			Annotations: referrer.GetAnnotations(),
			CreatedAt:   referrer.GetCreatedAt(),
			Data:        referrer.GetData(),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid2 := response2.GetReferrerRef().GetCid()
		gomega.Expect(cid2).ToNot(gomega.BeEmpty())

		gomega.Expect(cid1).ToNot(gomega.Equal(cid2))

		referrers1 := pullReferrers(c, ctx, record1)
		gomega.Expect(referrers1).To(gomega.HaveLen(1))

		referrers2 := pullReferrers(c, ctx, record2)
		gomega.Expect(referrers2).To(gomega.HaveLen(1))
	})

	ginkgo.It("should successfully push & pull referrer stream", func() {
		var err error

		// PUSH
		push, err := c.StoreServiceClient.PushReferrer(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		referrer1 := generateReferrer()
		referrer1.Annotations = map[string]string{"test_id": "1"}
		err = push.Send(&storev1.PushReferrerRequest{
			RecordRef:   record1,
			Type:        referrer1.GetType(),
			Annotations: referrer1.GetAnnotations(),
			CreatedAt:   referrer1.GetCreatedAt(),
			Data:        referrer1.GetData(),
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		referrer2 := generateReferrer()
		referrer2.Annotations = map[string]string{"test_id": "2"}
		err = push.Send(&storev1.PushReferrerRequest{
			RecordRef:   record2,
			Type:        referrer2.GetType(),
			Annotations: referrer2.GetAnnotations(),
			CreatedAt:   referrer2.GetCreatedAt(),
			Data:        referrer2.GetData(),
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = push.CloseSend()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response1, err := push.Recv()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(response1.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response1.GetErrorMessage()).To(gomega.BeEmpty())
		gomega.Expect(response1.GetReferrerRef().GetCid()).NotTo(gomega.BeNil())
		gomega.Expect(response1.GetReferrerRef().GetCid()).NotTo(gomega.BeEmpty())

		response2, err := push.Recv()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(response2.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response2.GetErrorMessage()).To(gomega.BeEmpty())
		gomega.Expect(response2.GetReferrerRef().GetCid()).NotTo(gomega.BeNil())
		gomega.Expect(response2.GetReferrerRef().GetCid()).NotTo(gomega.BeEmpty())

		response3, err := push.Recv()
		gomega.Expect(err).To(gomega.BeIdenticalTo(io.EOF))
		gomega.Expect(response3.GetSuccess()).To(gomega.BeFalse())
		gomega.Expect(response3.GetErrorMessage()).To(gomega.BeEmpty())
		gomega.Expect(response3.GetReferrerRef().GetCid()).To(gomega.BeEmpty())

		// PULL
		pull, err := c.StoreServiceClient.PullReferrer(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// referrer 1
			RecordRef:    record1,
			ReferrerType: new(corev1.PublicKeyReferrerType),
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// referrer 1
			RecordRef:    record1,
			ReferrerType: new(corev1.PublicKeyReferrerType),
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// referrer 2
			RecordRef:    record2,
			ReferrerType: new(corev1.PublicKeyReferrerType),
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// no results
			RecordRef:    record1,
			ReferrerType: new(corev1.SignatureReferrerType),
		})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.CloseSend()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response4, err := pull.Recv()
		gomega.Expect(response4.GetReferrer().GetAnnotations()["test_id"]).To(gomega.Equal("1"))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response5, err := pull.Recv()
		gomega.Expect(response5.GetReferrer().GetAnnotations()["test_id"]).To(gomega.Equal("1"))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response6, err := pull.Recv()
		gomega.Expect(response6.GetReferrer().GetAnnotations()["test_id"]).To(gomega.Equal("2"))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response7, err := pull.Recv()
		gomega.Expect(response7.GetReferrer()).To(gomega.BeNil())
		gomega.Expect(err).To(gomega.BeIdenticalTo(io.EOF))
	})

	ginkgo.DescribeTable("validation errors",
		func(request *storev1.PushReferrerRequest, msg string) {
			_, err := c.PushReferrer(ctx, request)
			expectError(err, codes.InvalidArgument, msg)
		},
		ginkgo.Entry(
			"empty",
			&storev1.PushReferrerRequest{},
			"validation errors:\n"+
				" - record_ref: value is required\n"+
				" - type: value is required",
		),
		ginkgo.Entry(
			"record_ref: nil",
			&storev1.PushReferrerRequest{
				RecordRef: nil,
				Type:      corev1.PublicKeyReferrerType,
			},
			"validation error: record_ref: value is required",
		),
		ginkgo.Entry(
			"record_ref: empty",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{},
				Type:      corev1.PublicKeyReferrerType,
			},
			"validation error: record_ref.cid: value is required",
		),
		ginkgo.Entry(
			"record_ref: \"\"",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{Cid: ""},
				Type:      corev1.PublicKeyReferrerType,
			},
			"validation error: record_ref.cid: value is required",
		),
		ginkgo.Entry(
			"record_ref: too long",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{Cid: strings.Repeat("x", 129)},
				Type:      corev1.PublicKeyReferrerType,
			},
			"validation error: record_ref.cid: value length must be at most 128 characters",
		),
		ginkgo.Entry(
			"type: invalid",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{Cid: "foo"},
				Type:      "bar",
			},
			"validation error: type: value must be in list "+
				"[agntcy.dir.sign.v1.PublicKey, agntcy.dir.sign.v1.Signature]",
		),
	)
})
