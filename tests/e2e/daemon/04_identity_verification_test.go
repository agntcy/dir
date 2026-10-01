// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/multiformats/go-multibase"
	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// The reconciler's identity task runs every 10s in this environment (see
// testenv/default/dir-daemon-config.yaml), so a claim is checked within one or
// two ticks of being pushed.
const (
	claimVerificationTimeout = 90 * time.Second
	claimVerificationPoll    = 3 * time.Second
)

var _ = ginkgo.Describe("Identity claim verification", ginkgo.Ordered, ginkgo.Serial, func() {
	var (
		goodKey, otherKey ed25519.PrivateKey
		goodSubject       string
		otherSubject      string

		verifiedCID string
		failedCID   string
	)

	// didKey is the did:key subject of a key: it carries the key itself, so
	// verifying it needs no network.
	didKey := func(pub ed25519.PublicKey) string {
		encoded, err := multibase.Encode(multibase.Base58BTC, append([]byte{0xed, 0x01}, pub...))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		return "did:key:" + encoded
	}

	signerFor := func(key ed25519.PrivateKey) jws.Signer {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		signer, err := jws.NewKeySigner(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		return signer
	}

	// pushRecord pushes a record under a fresh name, declaring annotations.
	pushRecord := func(ctx context.Context, annotations map[string]string) string {
		var record map[string]any
		gomega.Expect(json.Unmarshal(testdata.ExpectedRecordV080V4JSON, &record)).To(gomega.Succeed())

		nonce := make([]byte, 6)
		_, err := rand.Read(nonce)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		record["name"] = "example.com/identity-e2e/" + hex.EncodeToString(nonce)
		record["annotations"] = annotations

		data, err := json.Marshal(record)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		parsed, err := corev1.UnmarshalRecord(data)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		ref, err := testEnv.Client.Push(ctx, parsed)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		return ref.GetCid()
	}

	statusOf := func(ctx context.Context, cid string) *identityv1.GetIdentityStatusResponse {
		ginkgo.GinkgoHelper()

		resp, err := testEnv.Client.GetIdentityStatus(ctx, cid)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		return resp
	}

	searchByClaims := func(ctx context.Context, queries ...*searchv1.RecordQuery) []string {
		ginkgo.GinkgoHelper()

		result, err := testEnv.Client.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{Queries: queries})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		var cids []string

		for {
			select {
			case resp := <-result.ResCh():
				if resp != nil {
					cids = append(cids, resp.GetRecordCid())
				}
			case <-result.ErrCh():
			case <-result.DoneCh():
				return cids
			}
		}
	}

	ginkgo.BeforeAll(func(ctx context.Context) {
		var err error

		_, goodKey, err = ed25519.GenerateKey(rand.Reader)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		_, otherKey, err = ed25519.GenerateKey(rand.Reader)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		goodSubject = didKey(goodKey.Public().(ed25519.PublicKey))   //nolint:forcetypeassert // ed25519 keys always are
		otherSubject = didKey(otherKey.Public().(ed25519.PublicKey)) //nolint:forcetypeassert // ed25519 keys always are
	})

	ginkgo.It("should verify a claim signed with the key its subject publishes", func(ctx context.Context) {
		verifiedCID = pushRecord(ctx, map[string]string{corev1.AnnotationKeyIdentity: goodSubject})

		_, err := testEnv.Client.ClaimIdentity(ctx, verifiedCID, signerFor(goodKey))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Eventually(func(g gomega.Gomega) {
			identity := statusOf(ctx, verifiedCID).GetIdentity()

			g.Expect(identity).NotTo(gomega.BeNil(), "not verified yet")
			g.Expect(identity.GetStatus()).To(gomega.Equal(identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED))
			g.Expect(identity.GetSubject()).To(gomega.Equal(goodSubject))
			g.Expect(identity.GetRole()).To(gomega.Equal(identityv1.ClaimRole_CLAIM_ROLE_IDENTITY))
		}).WithContext(ctx).WithTimeout(claimVerificationTimeout).WithPolling(claimVerificationPoll).Should(gomega.Succeed())

		gomega.Expect(statusOf(ctx, verifiedCID).GetOwner()).To(gomega.BeNil(), "no ownership claim was pushed")
	})

	ginkgo.It("should fail a claim signed with a key its subject does not publish", func(ctx context.Context) {
		// The record declares otherSubject, but the claim is signed by goodKey.
		failedCID = pushRecord(ctx, map[string]string{corev1.AnnotationKeyOwner: otherSubject})

		_, err := testEnv.Client.ClaimOwnership(ctx, failedCID, signerFor(goodKey))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Eventually(func(g gomega.Gomega) {
			owner := statusOf(ctx, failedCID).GetOwner()

			g.Expect(owner).NotTo(gomega.BeNil(), "not verified yet")
			g.Expect(owner.GetStatus()).To(gomega.Equal(identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_FAILED))
			g.Expect(owner.GetError()).NotTo(gomega.BeEmpty())
		}).WithContext(ctx).WithTimeout(claimVerificationTimeout).WithPolling(claimVerificationPoll).Should(gomega.Succeed())
	})

	ginkgo.It("should find the records by their verified claims", func(ctx context.Context) {
		verified := searchByClaims(ctx, &searchv1.RecordQuery{
			Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_IDENTITY_VERIFIED, Value: "true",
		})
		gomega.Expect(verified).To(gomega.ContainElement(verifiedCID))
		gomega.Expect(verified).NotTo(gomega.ContainElement(failedCID))

		bySubject := searchByClaims(ctx, &searchv1.RecordQuery{
			Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_IDENTITY, Value: goodSubject,
		})
		gomega.Expect(bySubject).To(gomega.ConsistOf(verifiedCID))

		// The owner claim failed, so the record has the subject but no verified owner.
		byOwner := searchByClaims(ctx, &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_OWNER, Value: otherSubject})
		gomega.Expect(byOwner).To(gomega.ConsistOf(failedCID))

		verifiedOwners := searchByClaims(ctx,
			&searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_OWNER, Value: otherSubject},
			&searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_OWNER_VERIFIED, Value: "true"},
		)
		gomega.Expect(verifiedOwners).To(gomega.BeEmpty())
	})

	ginkgo.It("should refuse a claim for a subject the record does not declare", func(ctx context.Context) {
		cid := pushRecord(ctx, map[string]string{})

		_, err := testEnv.Client.ClaimIdentity(ctx, cid, signerFor(goodKey))
		gomega.Expect(err).To(gomega.HaveOccurred())
	})
})
