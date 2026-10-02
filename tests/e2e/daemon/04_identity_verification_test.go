// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
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

	// A claim for a subject the record does not declare can only get in by being pushed
	// by hand. It leaves no result: its subject is chosen by whoever attached it, and
	// the subject filters would match it. A legitimate claim on the same record is the
	// marker that the task has been through the record.
	ginkgo.It("should not let a claim for another subject leave a result or reach the search index", func(ctx context.Context) {
		const spoofed = "dns:victim.example"

		cid := pushRecord(ctx, map[string]string{corev1.AnnotationKeyOwner: goodSubject})

		claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: spoofed}
		gomega.Expect(clientidentity.Sign(claim, cid, signerFor(goodKey))).To(gomega.Succeed())

		referrer, err := claim.MarshalReferrer()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		resp, err := testEnv.Client.PushReferrer(ctx, &storev1.PushReferrerRequest{
			RecordRef:   &corev1.RecordRef{Cid: cid},
			Type:        referrer.GetType(),
			Annotations: referrer.GetAnnotations(),
			CreatedAt:   referrer.GetCreatedAt(),
			Data:        referrer.GetData(),
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		gomega.Expect(resp.GetSuccess()).To(gomega.BeTrue())

		_, err = testEnv.Client.ClaimOwnership(ctx, cid, signerFor(goodKey))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Eventually(func(g gomega.Gomega) {
			owner := statusOf(ctx, cid).GetOwner()

			g.Expect(owner).NotTo(gomega.BeNil(), "not verified yet")
			g.Expect(owner.GetStatus()).To(gomega.Equal(identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED))
		}).WithContext(ctx).WithTimeout(claimVerificationTimeout).WithPolling(claimVerificationPoll).Should(gomega.Succeed())

		gomega.Expect(statusOf(ctx, cid).GetIdentity()).To(gomega.BeNil(), "the spoofed claim leaves no result")

		bySubject := searchByClaims(ctx, &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_IDENTITY, Value: spoofed})
		gomega.Expect(bySubject).NotTo(gomega.ContainElement(cid))
	})

	ginkgo.It("should reject a negated claim query and a false claim verified query", func(ctx context.Context) {
		for _, query := range []*searchv1.RecordQuery{
			{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_OWNER, Value: "dns:acme.com", Negate: true},
			{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_IDENTITY_VERIFIED, Value: "false"},
		} {
			result, err := testEnv.Client.SearchCIDs(ctx, &searchv1.SearchCIDsRequest{Queries: []*searchv1.RecordQuery{query}})
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			var failed bool

			for done := false; !done; {
				select {
				case <-result.ResCh():
				case err := <-result.ErrCh():
					failed = failed || err != nil
				case <-result.DoneCh():
					done = true
				}
			}

			gomega.Expect(failed).To(gomega.BeTrue(), query.String())
		}
	})

	ginkgo.It("should drop a result when its claim is withdrawn, and stop finding a deleted record", func(ctx context.Context) {
		referrerType := corev1.IdentityClaimReferrerType

		_, err := testEnv.Client.DeleteReferrer(ctx, &storev1.DeleteReferrerRequest{
			Record:       &corev1.RecordRef{Cid: verifiedCID},
			ReferrerType: &referrerType,
		})
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(statusOf(ctx, verifiedCID).GetIdentity()).To(gomega.BeNil())
		}).WithContext(ctx).WithTimeout(claimVerificationTimeout).WithPolling(claimVerificationPoll).Should(gomega.Succeed())

		verified := searchByClaims(ctx, &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_IDENTITY_VERIFIED, Value: "true"})
		gomega.Expect(verified).NotTo(gomega.ContainElement(verifiedCID))

		gomega.Expect(testEnv.Client.Delete(ctx, &corev1.RecordRef{Cid: failedCID})).To(gomega.Succeed())

		byOwner := searchByClaims(ctx, &searchv1.RecordQuery{Type: searchv1.RecordQueryType_RECORD_QUERY_TYPE_OWNER, Value: otherSubject})
		gomega.Expect(byOwner).To(gomega.BeEmpty())
	})

	// The bundle file is read on every run, so replacing it is how a trust domain
	// rotates or revokes its CA. The daemon configs point at spiffeBundlePath.
	ginkgo.It("should follow the SPIFFE trust bundle: verified, failed once it is replaced, verified once restored", func(ctx context.Context) {
		const subject = "spiffe://e2e.test/agents/lifecycle"

		trusted, other := newE2ECA("e2e trusted root"), newE2ECA("e2e other root")

		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		der, err := x509.MarshalPKCS8PrivateKey(key)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		signer, err := jws.NewKeySigner(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), nil)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		trusted.writeBundle()

		cid := pushRecord(ctx, map[string]string{corev1.AnnotationKeyIdentity: subject})

		_, err = testEnv.Client.ClaimIdentity(ctx, cid, signer, clientidentity.WithCertificate(trusted.issue(key, subject)))
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		expectIdentityStatus := func(want identityv1.ClaimVerificationStatus) {
			ginkgo.GinkgoHelper()

			gomega.Eventually(func(g gomega.Gomega) {
				identity := statusOf(ctx, cid).GetIdentity()

				g.Expect(identity).NotTo(gomega.BeNil(), "not verified yet")
				g.Expect(identity.GetStatus()).To(gomega.Equal(want))
			}).WithContext(ctx).WithTimeout(claimVerificationTimeout).WithPolling(claimVerificationPoll).Should(gomega.Succeed())
		}

		expectIdentityStatus(identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED)

		other.writeBundle()
		expectIdentityStatus(identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_FAILED)

		trusted.writeBundle()
		expectIdentityStatus(identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED)
	})
})

// spiffeBundlePath is where the daemon configs read the e2e.test trust bundle from.
const spiffeBundlePath = "/tmp/dir-e2e-identity/spiffe-bundle.pem"

// e2eCA is a throwaway certificate authority for SPIFFE SVIDs.
type e2eCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func newE2ECA(name string) *e2eCA {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	cert, err := x509.ParseCertificate(der)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return &e2eCA{cert: cert, key: key}
}

// writeBundle makes this CA the only one the daemon trusts for e2e.test.
func (ca *e2eCA) writeBundle() {
	gomega.Expect(os.MkdirAll(filepath.Dir(spiffeBundlePath), 0o755)).To(gomega.Succeed())

	bundle := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw})
	gomega.Expect(os.WriteFile(spiffeBundlePath, bundle, 0o600)).To(gomega.Succeed())
}

// issue returns an SVID for the SPIFFE ID over key's public half.
func (ca *e2eCA) issue(key *ecdsa.PrivateKey, spiffeID string) []byte {
	uri, err := url.Parse(spiffeID)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		URIs:         []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
