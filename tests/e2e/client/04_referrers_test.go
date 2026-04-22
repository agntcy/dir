// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:modernize
package client

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	"strings"
	"time"

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

var _ = ginkgo.Describe("Referrers E2E Tests", ginkgo.Label("referrers"), ginkgo.Ordered, ginkgo.Serial, func() {
	var (
		record1 *corev1.RecordRef
		record2 *corev1.RecordRef
	)

	ginkgo.BeforeEach(func(ctx context.Context) {
		var err error

		record1, err = testEnv.Client.Push(ctx, generateRecord())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		record2, err = testEnv.Client.Push(ctx, generateRecord())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func(ctx context.Context) {
		testEnv.Client.Delete(ctx, record1) //nolint:errcheck
		testEnv.Client.Delete(ctx, record2) //nolint:errcheck
	})

	ginkgo.It("should successfully push basic referrer", func(ctx context.Context) {
		referrer := generatePublicKeyReferrer()
		response, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, referrer))

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(response.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response.GetErrorMessage()).To(gomega.BeEmpty())
		referrerCID := response.GetReferrerRef().GetCid()
		gomega.Expect(referrerCID).NotTo(gomega.BeNil())
		gomega.Expect(referrerCID).NotTo(gomega.BeEmpty())

		referrers, err := pullReferrers(
			ctx,
			testEnv.Client,
			&storev1.PullReferrerRequest{
				RecordRef:    record1,
				ReferrerType: toPtr(corev1.PublicKeyReferrerType),
			},
		)

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetRecordRef().GetCid()).To(gomega.Equal(record1.GetCid()))
		gomega.Expect(referrers[0].GetType()).To(gomega.Equal(corev1.PublicKeyReferrerType))
		gomega.Expect(referrers[0].GetAnnotations()).To(gomega.BeNil())
		gomega.Expect(referrers[0].GetCreatedAt()).To(gomega.Equal(""))
		gomega.Expect(referrers[0].GetData().AsMap()).To(gomega.Equal(referrer.GetData().AsMap()))
		gomega.Expect(referrers[0].GetReferrerRef().GetCid()).To(gomega.Equal(referrerCID))
	})

	ginkgo.It("should successfully push full referrer", func(ctx context.Context) {
		referrer := generatePublicKeyReferrer()
		referrer.CreatedAt = "2026-03-09T14:20:00Z"
		referrer.RecordRef = record1
		referrer.Annotations = map[string]string{"foo": "bar"}
		response, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, referrer))

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(response.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response.GetErrorMessage()).To(gomega.BeEmpty())
		referrerCID := response.GetReferrerRef().GetCid()
		gomega.Expect(referrerCID).NotTo(gomega.BeNil())
		gomega.Expect(referrerCID).NotTo(gomega.BeEmpty())

		// Validate CID
		referrerBytes, err := referrer.Marshal()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		referrerDigest, err := corev1.CalculateDigest(referrerBytes)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		expectedCID, err := corev1.ConvertDigestToCID(referrerDigest)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(referrerCID).To(gomega.Equal(expectedCID))

		referrers, err := pullReferrers(
			ctx,
			testEnv.Client,
			&storev1.PullReferrerRequest{
				RecordRef:    record1,
				ReferrerType: toPtr(corev1.PublicKeyReferrerType),
			},
		)

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetRecordRef().GetCid()).To(gomega.Equal(record1.GetCid()))
		gomega.Expect(referrers[0].GetType()).To(gomega.Equal(corev1.PublicKeyReferrerType))
		gomega.Expect(referrers[0].GetAnnotations()).To(gomega.Equal(map[string]string{"foo": "bar"}))
		gomega.Expect(referrers[0].GetCreatedAt()).To(gomega.Equal("2026-03-09T14:20:00Z"))
		gomega.Expect(referrers[0].GetData().AsMap()).To(gomega.Equal(referrer.GetData().AsMap()))
		gomega.Expect(referrers[0].GetReferrerRef().GetCid()).To(gomega.Equal(referrerCID))
	})

	ginkgo.It("should pass if referrer exists", func(ctx context.Context) {
		referrer := generatePublicKeyReferrer()

		response1, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, referrer))

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid1 := response1.GetReferrerRef().GetCid()
		gomega.Expect(cid1).ToNot(gomega.BeEmpty())

		response2, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, referrer))

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid2 := response2.GetReferrerRef().GetCid()
		gomega.Expect(cid2).To(gomega.Equal(cid1))

		referrers, err := pullReferrers(
			ctx,
			testEnv.Client,
			&storev1.PullReferrerRequest{
				RecordRef:    record1,
				ReferrerType: toPtr(corev1.PublicKeyReferrerType),
			},
		)

		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetReferrerRef().GetCid()).To(gomega.Equal(cid1))
		gomega.Expect(referrers[0].GetReferrerRef().GetCid()).To(gomega.Equal(cid2))
	})

	ginkgo.It("should pass if same referrer different records", func(ctx context.Context) {
		referrer := generatePublicKeyReferrer()

		response1, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, referrer))

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid1 := response1.GetReferrerRef().GetCid()
		gomega.Expect(cid1).ToNot(gomega.BeEmpty())

		response2, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record2, referrer))

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		cid2 := response2.GetReferrerRef().GetCid()
		gomega.Expect(cid2).ToNot(gomega.BeEmpty())

		gomega.Expect(cid1).ToNot(gomega.Equal(cid2))

		referrers1, err := pullReferrers(
			ctx,
			testEnv.Client,
			&storev1.PullReferrerRequest{
				RecordRef:    record1,
				ReferrerType: toPtr(corev1.PublicKeyReferrerType),
			},
		)

		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(referrers1).To(gomega.HaveLen(1))
		gomega.Expect(referrers1[0].GetReferrerRef().GetCid()).To(gomega.Equal(cid1))

		referrers2, err := pullReferrers(
			ctx,
			testEnv.Client,
			&storev1.PullReferrerRequest{
				RecordRef:    record2,
				ReferrerType: toPtr(corev1.PublicKeyReferrerType),
			},
		)

		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(referrers2).To(gomega.HaveLen(1))
		gomega.Expect(referrers2[0].GetReferrerRef().GetCid()).To(gomega.Equal(cid2))
	})

	ginkgo.It("should successfully push & pull referrer stream", func(ctx context.Context) {
		var err error

		// PUSH
		push, err := testEnv.Client.StoreServiceClient.PushReferrer(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		referrer1 := generatePublicKeyReferrer()
		referrer1.Annotations = map[string]string{"test_id": "1"}
		err = push.Send(newPushReferrerRequest(record1, referrer1))

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		referrer2 := generatePublicKeyReferrer()
		referrer2.Annotations = map[string]string{"test_id": "2"}
		err = push.Send(newPushReferrerRequest(record2, referrer2))

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = push.CloseSend()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response1, err := push.Recv()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(response1.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response1.GetErrorMessage()).To(gomega.BeEmpty())

		cid1 := response1.GetReferrerRef().GetCid()
		gomega.Expect(cid1).NotTo(gomega.BeNil())
		gomega.Expect(cid1).NotTo(gomega.BeEmpty())

		response2, err := push.Recv()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(response2.GetSuccess()).To(gomega.BeTrue())
		gomega.Expect(response2.GetErrorMessage()).To(gomega.BeEmpty())

		cid2 := response2.GetReferrerRef().GetCid()
		gomega.Expect(cid2).NotTo(gomega.BeNil())
		gomega.Expect(cid2).NotTo(gomega.BeEmpty())
		gomega.Expect(cid2).NotTo(gomega.Equal(cid1))

		response3, err := push.Recv()
		gomega.Expect(err).To(gomega.BeIdenticalTo(io.EOF))
		gomega.Expect(response3.GetSuccess()).To(gomega.BeFalse())
		gomega.Expect(response3.GetErrorMessage()).To(gomega.BeEmpty())
		gomega.Expect(response3.GetReferrerRef().GetCid()).To(gomega.BeEmpty())

		// PULL
		pull, err := testEnv.Client.StoreServiceClient.PullReferrer(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// referrer 1
			RecordRef:    record1,
			ReferrerType: toPtr(corev1.PublicKeyReferrerType),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// referrer 1
			RecordRef:    record1,
			ReferrerType: toPtr(corev1.PublicKeyReferrerType),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// referrer 2
			RecordRef:    record2,
			ReferrerType: toPtr(corev1.PublicKeyReferrerType),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.Send(&storev1.PullReferrerRequest{
			// no results
			RecordRef:    record1,
			ReferrerType: toPtr(corev1.SignatureReferrerType),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		err = pull.CloseSend()
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response4, err := pull.Recv()
		gomega.Expect(response4.GetReferrer().GetAnnotations()["test_id"]).To(gomega.Equal("1"))
		gomega.Expect(response4.GetReferrer().GetReferrerRef().GetCid()).To(gomega.Equal(cid1))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response5, err := pull.Recv()
		gomega.Expect(response5.GetReferrer().GetAnnotations()["test_id"]).To(gomega.Equal("1"))
		gomega.Expect(response5.GetReferrer().GetReferrerRef().GetCid()).To(gomega.Equal(cid1))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response6, err := pull.Recv()
		gomega.Expect(response6.GetReferrer().GetAnnotations()["test_id"]).To(gomega.Equal("2"))
		gomega.Expect(response6.GetReferrer().GetReferrerRef().GetCid()).To(gomega.Equal(cid2))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		response7, err := pull.Recv()
		gomega.Expect(response7.GetReferrer()).To(gomega.BeNil())
		gomega.Expect(err).To(gomega.BeIdenticalTo(io.EOF))
	})

	ginkgo.DescribeTable("PullReferrer validation errors",
		func(ctx context.Context, request *storev1.PullReferrerRequest, msg string) {
			referrers, err := pullReferrers(ctx, testEnv.Client, request)
			expectError(err, codes.InvalidArgument, msg)
			gomega.Expect(referrers).To(gomega.BeEmpty())
		},
		ginkgo.Entry(
			"empty",
			&storev1.PullReferrerRequest{},
			"validation error: record_ref: value is required",
		),
		ginkgo.Entry(
			"record_ref: empty",
			&storev1.PullReferrerRequest{
				RecordRef: &corev1.RecordRef{},
			},
			"validation error: record_ref.cid: value is required",
		),
		ginkgo.Entry(
			"record_ref: \"\"",
			&storev1.PullReferrerRequest{
				RecordRef: &corev1.RecordRef{Cid: ""},
			},
			"validation error: record_ref.cid: value is required",
		),
		ginkgo.Entry(
			"referrer_type: invalid",
			&storev1.PullReferrerRequest{
				RecordRef:    &corev1.RecordRef{Cid: "foo"},
				ReferrerType: toPtr("bar"),
			},
			"validation error: referrer_type: value must be a valid referrer type",
		),
		ginkgo.Entry(
			"referrer_type: \"\"",
			&storev1.PullReferrerRequest{
				RecordRef:    &corev1.RecordRef{Cid: "foo"},
				ReferrerType: toPtr(""),
			},
			"validation error: referrer_type: value must be a valid referrer type",
		),
		ginkgo.Entry(
			"referrer_ref: empty",
			&storev1.PullReferrerRequest{
				RecordRef:   &corev1.RecordRef{Cid: "foo"},
				ReferrerRef: &corev1.ReferrerRef{},
			},
			"validation error: referrer_ref.cid: value is required",
		),
		ginkgo.Entry(
			"referrer_ref.cid: \"\"",
			&storev1.PullReferrerRequest{
				RecordRef:   &corev1.RecordRef{Cid: "foo"},
				ReferrerRef: &corev1.ReferrerRef{Cid: ""},
			},
			"validation error: referrer_ref.cid: value is required",
		),
	)

	ginkgo.DescribeTable("PushReferrer validation errors",
		func(ctx context.Context, request *storev1.PushReferrerRequest, msg string) {
			_, err := testEnv.Client.PushReferrer(ctx, request)
			expectError(err, codes.InvalidArgument, msg)
		},
		ginkgo.Entry(
			"empty",
			&storev1.PushReferrerRequest{},
			getPushReferrerError("validation errors:\n"+
				" - record_ref: value is required\n"+
				" - type: value is required"),
		),
		ginkgo.Entry(
			"record_ref: nil",
			&storev1.PushReferrerRequest{
				RecordRef: nil,
				Type:      corev1.PublicKeyReferrerType,
			},
			getPushReferrerError("validation error: record_ref: value is required"),
		),
		ginkgo.Entry(
			"record_ref: empty",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{},
				Type:      corev1.PublicKeyReferrerType,
			},
			getPushReferrerError("validation error: record_ref.cid: value is required"),
		),
		ginkgo.Entry(
			"record_ref: \"\"",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{Cid: ""},
				Type:      corev1.PublicKeyReferrerType,
			},
			getPushReferrerError("validation error: record_ref.cid: value is required"),
		),
		ginkgo.Entry(
			"record_ref: too long",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{Cid: strings.Repeat("x", 129)},
				Type:      corev1.PublicKeyReferrerType,
			},
			getPushReferrerError("validation error: record_ref.cid: value must be a valid CID"),
		),
		ginkgo.Entry(
			"type: invalid",
			&storev1.PushReferrerRequest{
				RecordRef: &corev1.RecordRef{Cid: "foo"},
				Type:      "bar",
			},
			getPushReferrerError("validation error: type: value must be a valid referrer type"),
		),
	)

	ginkgo.It("should successfully pull referrers", func(ctx context.Context) {
		referrer1 := generatePublicKeyReferrer()
		response1, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, referrer1))

		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		cid1 := response1.GetReferrerRef().GetCid()
		gomega.Expect(cid1).NotTo(gomega.BeEmpty())

		referrer2 := generatePublicKeyReferrer()
		response2, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, referrer2))

		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		cid2 := response2.GetReferrerRef().GetCid()
		gomega.Expect(cid2).NotTo(gomega.BeEmpty())

		referrers, err := pullReferrers(ctx, testEnv.Client, &storev1.PullReferrerRequest{RecordRef: record1})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(referrers).To(gomega.HaveLen(2))
		gomega.Expect(referrers[0].GetReferrerRef().GetCid()).To(gomega.Equal(cid1))
		gomega.Expect(referrers[1].GetReferrerRef().GetCid()).To(gomega.Equal(cid2))

		referrers, err = pullReferrers(ctx, testEnv.Client, &storev1.PullReferrerRequest{
			RecordRef:   record1,
			ReferrerRef: &corev1.ReferrerRef{Cid: cid1},
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetReferrerRef().GetCid()).To(gomega.Equal(cid1))

		referrers, err = pullReferrers(ctx, testEnv.Client, &storev1.PullReferrerRequest{
			RecordRef:   record1,
			ReferrerRef: &corev1.ReferrerRef{Cid: cid2},
		})

		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(referrers).To(gomega.HaveLen(1))
		gomega.Expect(referrers[0].GetReferrerRef().GetCid()).To(gomega.Equal(cid2))
	})
})

var _ = ginkgo.Describe("DeleteReferrer", func() {
	var (
		// records
		record1 *corev1.RecordRef
		record2 *corev1.RecordRef

		// referrers
		referrer1 *corev1.ReferrerRef // record 1, public key
		referrer2 *corev1.ReferrerRef // record 1, public key
		referrer3 *corev1.ReferrerRef // record 1, signature
		referrer4 *corev1.ReferrerRef // record 2, public key
	)

	ginkgo.BeforeEach(func(ctx context.Context) {
		var err error

		record1, err = testEnv.Client.Push(ctx, generateRecord())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		record2, err = testEnv.Client.Push(ctx, generateRecord())
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		response1, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, generatePublicKeyReferrer()))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		referrer1 = response1.GetReferrerRef()

		response2, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, generatePublicKeyReferrer()))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		referrer2 = response2.GetReferrerRef()

		response3, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record1, generateSignatureReferrer()))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		referrer3 = response3.GetReferrerRef()

		response4, err := testEnv.Client.PushReferrer(ctx, newPushReferrerRequest(record2, generatePublicKeyReferrer()))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		referrer4 = response4.GetReferrerRef()
	})

	ginkgo.AfterEach(func(ctx context.Context) {
		testEnv.Client.Delete(ctx, record1) //nolint:errcheck
		testEnv.Client.Delete(ctx, record2) //nolint:errcheck
	})

	ginkgo.It("delete & pull", func(ctx context.Context) {
		response1, err := testEnv.Client.DeleteReferrer(ctx, &storev1.DeleteReferrerRequest{
			Record:       &corev1.RecordRef{Cid: record1.GetCid()},
			ReferrerRef:  &corev1.ReferrerRef{Cid: referrer2.GetCid()},
			ReferrerType: toPtr(corev1.PublicKeyReferrerType),
		})

		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(response1.GetReferrerRefs()).To(gomega.HaveLen(1))
		gomega.Expect(response1.GetReferrerRefs()[0].GetCid()).To(gomega.Equal(referrer2.GetCid()))

		referrers1, err := pullReferrers(ctx, testEnv.Client, &storev1.PullReferrerRequest{RecordRef: record1})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(referrers1).To(gomega.HaveLen(2))
		gomega.Expect(referrers1[0].GetReferrerRef().GetCid()).To(gomega.Equal(referrer1.GetCid()))
		gomega.Expect(referrers1[1].GetReferrerRef().GetCid()).To(gomega.Equal(referrer3.GetCid()))

		referrers2, err := pullReferrers(ctx, testEnv.Client, &storev1.PullReferrerRequest{RecordRef: record2})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
		gomega.Expect(referrers2).To(gomega.HaveLen(1))
		gomega.Expect(referrers2[0].GetReferrerRef().GetCid()).To(gomega.Equal(referrer4.GetCid()))
	})

	ginkgo.It("record doesn't exist", func(ctx context.Context) {
		_, err := testEnv.Client.DeleteReferrer(ctx, &storev1.DeleteReferrerRequest{
			Record: &corev1.RecordRef{Cid: "foo"},
		})

		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.DescribeTable("delete",
		func(ctx context.Context, args func() (*storev1.DeleteReferrerRequest, *[]string)) {
			request, expectedCIDs := args()

			response, err := testEnv.Client.DeleteReferrer(ctx, request)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			cids := getCids(response.GetReferrerRefs())
			gomega.Expect(cids).To(gomega.Equal(expectedCIDs))
		},
		ginkgo.Entry(
			"by record CID (record 1)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}

				return r, &[]string{referrer1.GetCid(), referrer2.GetCid(), referrer3.GetCid()}
			},
		),
		ginkgo.Entry(
			"by record CID (record 2)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}

				return r, &[]string{referrer4.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer type (record 1, public key)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerType = toPtr(corev1.PublicKeyReferrerType)

				return r, &[]string{referrer1.GetCid(), referrer2.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer type (record 1, signature)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerType = toPtr(corev1.SignatureReferrerType)

				return r, &[]string{referrer3.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer type (record 2, public key)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}
				r.ReferrerType = toPtr(corev1.PublicKeyReferrerType)

				return r, &[]string{referrer4.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer type (record 2, signature)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}
				r.ReferrerType = toPtr(corev1.SignatureReferrerType)

				return r, &[]string{}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 1, referrer 1)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer1.GetCid()}

				return r, &[]string{referrer1.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 1, referrer 2)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer2.GetCid()}

				return r, &[]string{referrer2.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 1, referrer 3)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer3.GetCid()}

				return r, &[]string{referrer3.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 1, referrer 4)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer4.GetCid()}

				return r, &[]string{}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 2, referrer 1)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer1.GetCid()}

				return r, &[]string{}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 2, referrer 2)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer2.GetCid()}

				return r, &[]string{}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 2, referrer 3)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer3.GetCid()}

				return r, &[]string{}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 2, referrer 4)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer4.GetCid()}

				return r, &[]string{referrer4.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer CID (record 1, referrer doesn't exist)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record2.GetCid()}
				r.ReferrerRef = &corev1.ReferrerRef{Cid: "foo"}

				return r, &[]string{}
			},
		),
		ginkgo.Entry(
			"by referrer type & referrer CID (record 1, public key, referrer 1)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerType = toPtr(corev1.PublicKeyReferrerType)
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer1.GetCid()}

				return r, &[]string{referrer1.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer type & referrer CID (record 1, public key, referrer 2)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerType = toPtr(corev1.PublicKeyReferrerType)
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer2.GetCid()}

				return r, &[]string{referrer2.GetCid()}
			},
		),
		ginkgo.Entry(
			"by referrer type & referrer CID (record 1, public key, referrer 3)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerType = toPtr(corev1.PublicKeyReferrerType)
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer3.GetCid()}

				return r, &[]string{}
			},
		),
		ginkgo.Entry(
			"by referrer type & referrer CID (record 1, public key, referrer 4)",
			func() (*storev1.DeleteReferrerRequest, *[]string) {
				r := &storev1.DeleteReferrerRequest{}
				r.Record = &corev1.RecordRef{Cid: record1.GetCid()}
				r.ReferrerType = toPtr(corev1.PublicKeyReferrerType)
				r.ReferrerRef = &corev1.ReferrerRef{Cid: referrer4.GetCid()}

				return r, &[]string{}
			},
		),
	)

	ginkgo.DescribeTable("validation error",
		func(ctx context.Context, request *storev1.DeleteReferrerRequest, msg string) {
			_, err := testEnv.Client.DeleteReferrer(ctx, request)
			expectError(err, codes.InvalidArgument, msg)
		},
		ginkgo.Entry(
			"if request is empty",
			&storev1.DeleteReferrerRequest{},
			getDeleteReferrerError("validation error: record: value is required"),
		),
		ginkgo.Entry(
			"if record_ref: empty",
			&storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{}},
			getDeleteReferrerError("validation error: record.cid: value is required"),
		),
		ginkgo.Entry(
			"if record_ref.cid: \"\"",
			&storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{Cid: ""}},
			getDeleteReferrerError("validation error: record.cid: value is required"),
		),
		ginkgo.Entry(
			"if record_ref.cid: too long",
			&storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{Cid: strings.Repeat("a", 129)}},
			getDeleteReferrerError("validation error: record.cid: value must be a valid CID"),
		),
		ginkgo.Entry(
			"if referrer_ref: empty",
			&storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{Cid: "foo"}, ReferrerRef: &corev1.ReferrerRef{}},
			getDeleteReferrerError("validation error: referrer_ref.cid: value is required"),
		),
		ginkgo.Entry(
			"if referrer_ref.cid: \"\"",
			&storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{Cid: "foo"}, ReferrerRef: &corev1.ReferrerRef{Cid: ""}},
			getDeleteReferrerError("validation error: referrer_ref.cid: value is required"),
		),
		ginkgo.Entry(
			"if referrer_ref.cid: too long",
			&storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{Cid: "foo"}, ReferrerRef: &corev1.ReferrerRef{Cid: strings.Repeat("a", 129)}},
			getDeleteReferrerError("validation error: referrer_ref.cid: value must be a valid CID"),
		),
		ginkgo.Entry(
			"if referrer_type: invalid",
			&storev1.DeleteReferrerRequest{Record: &corev1.RecordRef{Cid: "foo"}, ReferrerType: toPtr("bar")},
			getDeleteReferrerError("validation error: referrer_type: value must be a valid referrer type"),
		),
	)
})

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

func generateSignature() string {
	_uuid := uuid.New()

	return base64.StdEncoding.EncodeToString(_uuid[:])
}

func generateRecord() *corev1.Record {
	record, err := corev1.UnmarshalRecord(testdata.ExpectedRecordV100JSON)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	record.Data.Fields["name"] = structpb.NewStringValue(record.GetName() + "_" + uuid.NewString()[:8])

	return record
}

func generatePublicKeyReferrer() *corev1.RecordReferrer {
	publicKey := signv1.PublicKey{Key: generatePublicKey()}
	referrer, err := publicKey.MarshalReferrer()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return referrer
}

func generateSignatureReferrer() *corev1.RecordReferrer {
	signature := signv1.Signature{
		SignedAt:  time.Now().UTC().Format(time.RFC3339),
		Signature: generateSignature(),
		Algorithm: "unknown",
	}

	referrer, err := signature.MarshalReferrer()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return referrer
}

func newPushReferrerRequest(
	ref *corev1.RecordRef,
	referrer *corev1.RecordReferrer,
) *storev1.PushReferrerRequest {
	return &storev1.PushReferrerRequest{
		RecordRef:   ref,
		Type:        referrer.GetType(),
		Annotations: referrer.GetAnnotations(),
		CreatedAt:   referrer.GetCreatedAt(),
		Data:        referrer.GetData(),
	}
}

func pullReferrers(
	ctx context.Context,
	c *client.Client,
	request *storev1.PullReferrerRequest,
) ([]*corev1.RecordReferrer, error) {
	responses, err := c.PullReferrer(ctx, request)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	referrers := []*corev1.RecordReferrer{}
	for _, response := range responses {
		referrers = append(referrers, response.GetReferrer())
	}

	return referrers, nil
}

func expectError(err error, code codes.Code, msg string) {
	gomega.Expect(err).To(gomega.HaveOccurred())
	e, ok := status.FromError(err)
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(e.Code()).To(gomega.Equal(code))
	gomega.Expect(e.Message()).To(gomega.Equal(msg))
}

func getPushReferrerError(desc string) string {
	return "failed to receive push referrer response: " +
		fmt.Sprintf("rpc error: code = InvalidArgument desc = %s", desc)
}

func getDeleteReferrerError(desc string) string {
	return "failed to receive delete referrer response: " +
		fmt.Sprintf("rpc error: code = InvalidArgument desc = %s", desc)
}

type WithCid interface {
	GetCid() string
}

func getCids[T WithCid](objs []T) *[]string {
	cids := []string{}

	for _, obj := range objs {
		cids = append(cids, obj.GetCid())
	}

	return &cids
}

func toPtr[T any](v T) *T {
	return &v
}
