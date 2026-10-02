// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/url"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ = ginkgo.Describe("Identity claims", ginkgo.Ordered, ginkgo.Serial, func() {
	const (
		recordName    = "directory.agntcy.org/cisco/name-resolution-test"
		recordVersion = "v1.0.0"

		identitySubject = "did:web:acme.com:agents:name-resolution"
		ownerSubject    = "dns:acme.com"
		spiffeSubject   = "spiffe://acme.com/agents/name-resolution"
	)

	var (
		recordCID string
		spiffeCID string
		key       *ecdsa.PrivateKey
		signer    jws.Signer
	)

	// pushRecord pushes the record in raw with the given annotations, which
	// declare the subjects a claim can then be made for.
	pushRecord := func(ctx context.Context, raw []byte, name string, annotations map[string]string) string {
		ginkgo.GinkgoHelper()

		var fields map[string]any
		gomega.Expect(json.Unmarshal(raw, &fields)).To(gomega.Succeed())

		if name != "" {
			fields["name"] = name
		}

		fields["annotations"] = annotations

		data, err := json.Marshal(fields)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		record, err := corev1.UnmarshalRecord(data)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ref, err := testEnv.Client.Push(ctx, record)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		return ref.GetCid()
	}

	ginkgo.BeforeAll(func(ctx context.Context) {
		recordCID = pushRecord(ctx, testdata.ExpectedRecordV070NameResolutionJSON, "", map[string]string{
			corev1.AnnotationKeyIdentity: identitySubject,
			corev1.AnnotationKeyOwner:    ownerSubject,
		})

		spiffeCID = pushRecord(ctx, testdata.ExpectedRecordV080V4JSON, "example.com/identity-e2e/spiffe", map[string]string{
			corev1.AnnotationKeyIdentity: spiffeSubject,
		})

		var err error

		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		der, err := x509.MarshalPKCS8PrivateKey(key)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		signer, err = jws.NewKeySigner(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	// pushedClaims returns the claims stored on the record under a referrer type.
	pushedClaims := func(ctx context.Context, cid, referrerType string) []*identityv1.Claim {
		ginkgo.GinkgoHelper()

		responses, err := testEnv.Client.PullReferrer(ctx, &storev1.PullReferrerRequest{
			RecordRef:    &corev1.RecordRef{Cid: cid},
			ReferrerType: &referrerType,
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		claims := make([]*identityv1.Claim, 0, len(responses))

		for _, response := range responses {
			claim := &identityv1.Claim{}
			gomega.Expect(claim.UnmarshalReferrer(response.GetReferrer())).To(gomega.Succeed())

			claims = append(claims, claim)
		}

		return claims
	}

	// signedByTestKey returns the stored claims about subject that were signed by
	// this run's key. The record may carry claims of earlier runs, signed by
	// other keys, which this leaves out.
	signedByTestKey := func(cid string, claims []*identityv1.Claim, subject string) []*identityv1.Claim {
		var mine []*identityv1.Claim

		for _, claim := range claims {
			if ok, _ := identity.Verify(claim, cid, subject, &key.PublicKey); ok {
				mine = append(mine, claim)
			}
		}

		return mine
	}

	ginkgo.It("should store an identity claim signed with a key alone", func(ctx context.Context) {
		claim, err := testEnv.Client.ClaimIdentity(ctx, recordCID, signer)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(claim.GetSubject()).To(gomega.Equal(identitySubject), "the subject is the one the record declares")

		// What was stored is a genuine claim for this record, signed by this key.
		claims := signedByTestKey(recordCID, pushedClaims(ctx, recordCID, corev1.IdentityClaimReferrerType), identitySubject)
		gomega.Expect(claims).To(gomega.HaveLen(1))
		gomega.Expect(claims[0].GetRole()).To(gomega.Equal(identityv1.ClaimRole_CLAIM_ROLE_IDENTITY))
		gomega.Expect(claims[0].Certificate).To(gomega.BeNil())
	})

	ginkgo.It("should store an ownership claim", func(ctx context.Context) {
		claim, err := testEnv.Client.ClaimOwnership(ctx, recordCID, signer)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(claim.GetSubject()).To(gomega.Equal(ownerSubject))

		claims := signedByTestKey(recordCID, pushedClaims(ctx, recordCID, corev1.OwnershipClaimReferrerType), ownerSubject)
		gomega.Expect(claims).To(gomega.HaveLen(1))
		gomega.Expect(claims[0].GetRole()).To(gomega.Equal(identityv1.ClaimRole_CLAIM_ROLE_OWNER))
	})

	ginkgo.It("should store a SPIFFE claim with its certificate", func(ctx context.Context) {
		uri, err := url.Parse(spiffeSubject)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: spiffeSubject},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			URIs:         []*url.URL{uri},
		}

		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		cert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

		_, err = testEnv.Client.ClaimIdentity(ctx, spiffeCID, signer)
		gomega.Expect(err).To(gomega.HaveOccurred(), "a spiffe:// subject needs its certificate")

		_, err = testEnv.Client.ClaimIdentity(ctx, spiffeCID, signer, identity.WithCertificate(cert))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		claims := signedByTestKey(spiffeCID, pushedClaims(ctx, spiffeCID, corev1.IdentityClaimReferrerType), spiffeSubject)
		gomega.Expect(claims).To(gomega.HaveLen(1))
		gomega.Expect(claims[0].GetCertificate()).NotTo(gomega.BeEmpty())
	})

	ginkgo.It("should refuse a claim for a record that does not exist", func(ctx context.Context) {
		_, err := testEnv.Client.ClaimIdentity(ctx, "baeareiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", signer)
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should refuse a claim the record does not declare a subject for", func(ctx context.Context) {
		_, err := testEnv.Client.ClaimOwnership(ctx, spiffeCID, signer)
		gomega.Expect(err).To(gomega.HaveOccurred())
	})

	ginkgo.It("should report no result for claims that have not been verified yet", func(ctx context.Context) {
		resp, err := testEnv.Client.GetIdentityStatus(ctx, recordCID)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "an unverified claim is not an error")
		gomega.Expect(resp).NotTo(gomega.BeNil())
	})

	ginkgo.It("should answer NotFound, not an empty status, for a CID that is not a record", func(ctx context.Context) {
		_, err := testEnv.Client.GetIdentityStatus(ctx, "baeareiaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		gomega.Expect(status.Code(err)).To(gomega.Equal(codes.NotFound))
	})

	ginkgo.It("should resolve the record by name", func(ctx context.Context) {
		resp, err := testEnv.Client.Resolve(ctx, recordName, "")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.GetRecords()).NotTo(gomega.BeEmpty())
		gomega.Expect(resp.GetRecords()[0].GetCid()).To(gomega.Equal(recordCID))
		gomega.Expect(resp.GetRecords()[0].GetName()).To(gomega.Equal(recordName))

		resp, err = testEnv.Client.Resolve(ctx, recordName, recordVersion)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.GetRecords()).To(gomega.HaveLen(1))
		gomega.Expect(resp.GetRecords()[0].GetVersion()).To(gomega.Equal(recordVersion))

		_, err = testEnv.Client.Resolve(ctx, recordName, "v99.0.0")
		gomega.Expect(err).To(gomega.HaveOccurred())
	})
})
