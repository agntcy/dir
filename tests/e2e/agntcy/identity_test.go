// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agntcy_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/google/uuid"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func (f *liveFixture) privateKey(name string) *rsa.PrivateKey {
	ginkgo.GinkgoHelper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.write(name, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded}))

	return key
}

func (f *liveFixture) publicKey(name string, key *rsa.PublicKey) {
	ginkgo.GinkgoHelper()

	encoded, err := x509.MarshalPKIXPublicKey(key)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.write(name, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}))
}

func (f *liveFixture) certificate() {
	ginkgo.GinkgoHelper()

	key := f.privateKey("tls.key")
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	template := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: "Directory E2E fixture"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(24 * time.Hour),
		IsCA: true, BasicConstraintsValid: true,
		KeyUsage:    x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{"host.docker.internal", "verifier"},
	}
	data, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.write("tls.crt", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: data}))
}

func (f *liveFixture) enroll(ctx context.Context) {
	ginkgo.GinkgoHelper()

	form := url.Values{"grant_type": {"client_credentials"}, "client_id": {"primary-client"}, "client_secret": {"disposable-client-secret"}}
	data, err := f.fetch(ctx, http.MethodPost,
		f.hostURL("keycloak", "http", "127.0.0.1")+"/realms/primary/protocol/openid-connect/token",
		"application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	var token struct {
		AccessToken string `json:"access_token"`
	}
	gomega.Expect(json.Unmarshal(data, &token)).To(gomega.Succeed())
	parts := strings.Split(token.AccessToken, ".")
	gomega.Expect(parts).To(gomega.HaveLen(3))
	claims, err := base64.RawURLEncoding.DecodeString(parts[1])
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	// Only extract a fixture identifier here. Real Node validates the OIDC proof.
	var identity struct {
		Subject string `json:"sub"`
	}
	gomega.Expect(json.Unmarshal(claims, &identity)).To(gomega.Succeed())
	gomega.Expect(identity.Subject).NotTo(gomega.BeEmpty())
	f.agentID = "IDP-" + identity.Subject
	f.subject = "agntcy://" + f.agentID
	f.agentKey = f.privateKey("agent.key")
	public := map[string]string{
		"kty": "RSA", "kid": "e2e-signer", "use": "sig", "alg": "RS256",
		"n": base64.RawURLEncoding.EncodeToString(f.agentKey.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(f.agentKey.E)).Bytes()),
	}
	issuer := map[string]any{"commonName": "host.docker.internal", "organization": "e2e", "publicKey": public, "authType": "ISSUER_AUTH_TYPE_IDP"}
	f.proof = map[string]string{"type": "JWT", "proofValue": token.AccessToken}
	input := map[string]any{"issuer": issuer, "proof": f.proof}
	node := f.hostURL("node", "http", "127.0.0.1")

	gomega.Eventually(ctx, func(ctx context.Context) error {
		return f.request(ctx, http.MethodPost, node+"/v1alpha1/issuer/register", input, nil)
	}, startupTimeout, time.Second).Should(gomega.Succeed(), "real Node enrollment")
	gomega.Expect(f.request(ctx, http.MethodPost, node+"/v1alpha1/id/generate", input, nil)).To(gomega.Succeed())
}

func (f *liveFixture) checkIdentity(ctx context.Context) {
	ginkgo.GinkgoHelper()

	var result struct {
		ResolverMetadata struct {
			ID string `json:"id"`
		} `json:"resolverMetadata"`
	}
	gomega.Expect(f.request(ctx, http.MethodPost, f.hostURL("node", "http", "127.0.0.1")+"/v1alpha1/id/resolve", map[string]string{"id": f.agentID}, &result)).To(gomega.Succeed())
	gomega.Expect(result.ResolverMetadata.ID).To(gomega.Equal(f.agentID))
}

func (f *liveFixture) push(ctx context.Context, name string, badgeLifetime time.Duration) string {
	ginkgo.GinkgoHelper()

	var record map[string]any
	gomega.Expect(json.Unmarshal(testdata.ExpectedRecordV080V4JSON, &record)).To(gomega.Succeed())
	record["name"] = "example.com/agntcy-e2e/" + name
	record["annotations"] = map[string]string{"agntcy.dir/identity": f.subject}
	data, err := json.Marshal(record)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	f.write(name+".json", data)
	output := f.mustCLI(ctx, "push", "/fixtures/"+name+".json", "--output", "json")

	var cid string
	if err := json.Unmarshal([]byte(output), &cid); err != nil {
		var result struct {
			CID string `json:"cid"`
		}
		gomega.Expect(json.Unmarshal([]byte(output), &result)).To(gomega.Succeed())
		cid = result.CID
	}

	gomega.Expect(cid).NotTo(gomega.BeEmpty())

	if badgeLifetime != 0 {
		f.publishBadge(ctx, record, badgeLifetime)
	}

	return cid
}

func (f *liveFixture) publishBadge(ctx context.Context, record map[string]any, lifetime time.Duration) {
	ginkgo.GinkgoHelper()

	now := time.Now().UTC().Truncate(time.Second)
	contextURLs := []string{"https://www.w3.org/2018/credentials/v1"}
	vc := map[string]any{
		"@context": contextURLs, "context": contextURLs, "type": []string{"VerifiableCredential", "AgentBadge"},
		"id": "urn:uuid:" + uuid.NewString(), "issuer": "host.docker.internal",
		"issuanceDate": now.Format(time.RFC3339), "expirationDate": now.Add(lifetime).Format(time.RFC3339),
		"credentialSubject": map[string]any{"id": f.agentID, "badge": record},
	}
	envelope := map[string]string{"envelopeType": "CREDENTIAL_ENVELOPE_TYPE_JOSE", "value": f.signBadge(vc)}

	var verified struct {
		Status bool `json:"status"`
	}

	node := f.hostURL("node", "http", "127.0.0.1")
	gomega.Expect(f.request(ctx, http.MethodPost, node+"/v1alpha1/vc/verify", map[string]any{"vc": envelope}, &verified)).To(gomega.Succeed())
	gomega.Expect(verified.Status).To(gomega.BeTrue(), "real Node must accept the badge signature")
	gomega.Expect(f.request(ctx, http.MethodPost, node+"/v1alpha1/vc/publish", map[string]any{"vc": envelope, "proof": f.proof}, nil)).To(gomega.Succeed())
}

func (f *liveFixture) signBadge(vc map[string]any) string {
	ginkgo.GinkgoHelper()

	header, err := json.Marshal(map[string]string{"alg": "RS256", "kid": "e2e-signer", "typ": "JOSE"})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	payload, err := json.Marshal(vc)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, f.agentKey, crypto.SHA256, digest[:])
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return input + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func (f *liveFixture) claim(ctx context.Context, cid, key string) {
	ginkgo.GinkgoHelper()
	f.mustCLI(ctx, "identity", "claim", "--record", cid, "--role", "identity", "--key", "/fixtures/"+key)
}

func (f *liveFixture) waitStatus(ctx context.Context, cid, expected string) {
	ginkgo.GinkgoHelper()
	gomega.Eventually(ctx, func(ctx context.Context) (string, error) {
		output, err := f.cli(ctx, "identity", "status", cid, "--output", "json")
		if err != nil {
			return "", err
		}

		var result struct {
			Identity struct {
				Status string `json:"status"`
			} `json:"identity"`
		}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			return "", fmt.Errorf("decode identity status: %w", err)
		}

		return result.Identity.Status, nil
	}, startupTimeout, time.Second).Should(gomega.Equal(expected))
}

func (f *liveFixture) search(ctx context.Context) string {
	ginkgo.GinkgoHelper()

	return f.mustCLI(ctx, "search", "--identity", f.subject, "--identity-verified")
}

func (f *liveFixture) checkStorage(ctx context.Context, cid string) {
	ginkgo.GinkgoHelper()
	gomega.Expect(f.mustCLI(ctx, "pull", cid, "--output", "json")).NotTo(gomega.BeEmpty())
	registry := f.hostURL("oci", "http", "127.0.0.1") + "/v2/dir"
	data, err := f.fetch(ctx, http.MethodGet, registry+"/manifests/"+cid, "application/json", nil)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(data))

	var index struct {
		Manifests []struct {
			Digest string `json:"digest"`
		} `json:"manifests"`
	}
	gomega.Expect(f.request(ctx, http.MethodGet, registry+"/referrers/"+digest, nil, &index)).To(gomega.Succeed())

	for _, ref := range index.Manifests {
		var manifest struct {
			Annotations map[string]string `json:"annotations"`
			Subject     struct {
				Digest string `json:"digest"`
			} `json:"subject"`
			Layers []struct {
				Digest string `json:"digest"`
			} `json:"layers"`
		}
		gomega.Expect(f.request(ctx, http.MethodGet, registry+"/manifests/"+ref.Digest, nil, &manifest)).To(gomega.Succeed())

		for _, value := range manifest.Annotations {
			if value != "agntcy.dir.identity.v1.IdentityClaim" {
				continue
			}

			gomega.Expect(manifest.Subject.Digest).To(gomega.Equal(digest))
			gomega.Expect(manifest.Layers).NotTo(gomega.BeEmpty())

			var blob struct {
				Data struct {
					RecordCID string `json:"recordCid"`
					Subject   string `json:"subject"`
				} `json:"data"`
			}
			gomega.Expect(f.request(ctx, http.MethodGet, registry+"/blobs/"+manifest.Layers[0].Digest, nil, &blob)).To(gomega.Succeed())
			gomega.Expect(blob.Data.RecordCID).To(gomega.Equal(cid))
			gomega.Expect(blob.Data.Subject).To(gomega.Equal(f.subject))

			return
		}
	}

	ginkgo.Fail("native IdentityClaim OCI manifest not found")
}
