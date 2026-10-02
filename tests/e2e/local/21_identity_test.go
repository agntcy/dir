// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/multiformats/go-multibase"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// identityStatusOutput is the JSON shape printed by `dirctl identity status --output json`.
type identityStatusOutput struct {
	CID      string         `json:"cid"`
	Identity map[string]any `json:"identity"`
	Owner    map[string]any `json:"owner"`
}

// identityResolveOutput is one entry of `dirctl identity resolve --output json`.
type identityResolveOutput struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	CID     string `json:"cid"`
}

var _ = ginkgo.Describe("Identity claims", ginkgo.Ordered, func() {
	const (
		recordName = "example.com/identity-e2e/research-assistant"

		spiffeSubject = "spiffe://acme.com/agents/research-assistant"
	)

	var (
		tempDir   string
		recordCID string
		keyPath   string
		certPath  string
		pushed    []string
	)

	// pushRecord pushes the test record under name, declaring annotations, and
	// returns its CID. An annotation declares the subject a claim is made for.
	pushRecord := func(name string, annotations map[string]string) string {
		ginkgo.GinkgoHelper()

		var fields map[string]any
		gomega.Expect(json.Unmarshal(testdata.ExpectedRecordV080V4JSON, &fields)).To(gomega.Succeed())

		if name != "" {
			fields["name"] = name
		}

		if annotations != nil {
			fields["annotations"] = annotations
		}

		data, err := json.Marshal(fields)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		path := filepath.Join(tempDir, strings.ReplaceAll(name, "/", "_")+".json")
		gomega.Expect(os.WriteFile(path, data, 0o600)).To(gomega.Succeed())

		cid := testEnv.CLI.Push(path).WithArgs("--output", "raw").ShouldSucceed()
		gomega.Expect(cid).NotTo(gomega.BeEmpty())

		pushed = append(pushed, cid)

		return cid
	}

	ginkgo.BeforeAll(func() {
		utils.ResetCLIState()

		var err error

		tempDir, err = os.MkdirTemp("", "identity-test")
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// The test record declares a SPIFFE identity and a DNS owner.
		recordCID = pushRecord(recordName, map[string]string{
			corev1.AnnotationKeyIdentity: spiffeSubject,
			corev1.AnnotationKeyOwner:    "dns:acme.com",
		})

		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		keyDER, err := x509.MarshalPKCS8PrivateKey(key)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		keyPath = filepath.Join(tempDir, "identity.key")
		gomega.Expect(os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}), 0o600)).To(gomega.Succeed())

		uri, err := url.Parse(spiffeSubject)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		template := &x509.Certificate{
			SerialNumber: big.NewInt(1),
			Subject:      pkix.Name{CommonName: spiffeSubject},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(time.Hour),
			URIs:         []*url.URL{uri},
		}

		certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		certPath = filepath.Join(tempDir, "identity.pem")
		gomega.Expect(os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}), 0o600)).To(gomega.Succeed())
	})

	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	ginkgo.AfterAll(func() {
		for _, cid := range pushed {
			_, _ = testEnv.CLI.Delete(cid).Execute()
		}

		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
	})

	claim := func(args ...string) *utils.CommandBuilder {
		return testEnv.CLI.Command("identity").WithArgs(append([]string{"claim"}, args...)...)
	}

	ginkgo.Context("claim", func() {
		// --cert is optional and independent of --key: a key alone is enough for
		// every subject that is not a SPIFFE ID.
		for _, subject := range []string{"dns:acme.com", "https://acme.com/agents", "did:web:acme.com:agents:research-assistant"} {
			ginkgo.It("should claim an identity with a key alone for "+subject, func() {
				cid := pushRecord("example.com/identity-e2e/"+strings.NewReplacer(":", "-", "/", "-").Replace(subject),
					map[string]string{corev1.AnnotationKeyIdentity: subject})

				output := claim("--record", cid, "--role", "identity", "--key", keyPath).ShouldSucceed()
				gomega.Expect(output).To(gomega.ContainSubstring(subject))
			})
		}

		ginkgo.It("should claim ownership by record name", func() {
			output := claim("--record", recordName, "--role", "owner", "--key", keyPath).
				WithArgs("--output", "json").
				ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			gomega.Expect(output).To(gomega.ContainSubstring("dns:acme.com"))
		})

		ginkgo.It("should claim a SPIFFE identity with a certificate", func() {
			_ = claim("--record", recordCID, "--role", "identity", "--key", keyPath, "--cert", certPath).
				ShouldSucceed()
		})

		ginkgo.It("should require a certificate for a SPIFFE subject", func() {
			_ = claim("--record", recordCID, "--role", "identity", "--key", keyPath).ShouldFail()
		})

		ginkgo.It("should refuse a certificate for any other subject", func() {
			_ = claim("--record", recordCID, "--role", "owner", "--key", keyPath, "--cert", certPath).
				ShouldFail()
		})

		ginkgo.It("should refuse a role the record declares no subject for", func() {
			cid := pushRecord("example.com/identity-e2e/no-owner", map[string]string{corev1.AnnotationKeyIdentity: "dns:acme.com"})

			_ = claim("--record", cid, "--role", "owner", "--key", keyPath).ShouldFail()
		})

		ginkgo.It("should no longer take the subject as a flag", func() {
			_ = claim("--record", recordCID, "--role", "owner", "--subject", "dns:acme.com", "--key", keyPath).ShouldFail()
		})

		ginkgo.It("should refuse an unknown role", func() {
			_ = claim("--record", recordCID, "--role", "admin", "--key", keyPath).ShouldFail()
		})

		ginkgo.It("should refuse a record that does not exist", func() {
			_ = claim("--record", "nonexistent.example.com/agent", "--role", "owner", "--key", keyPath).
				ShouldFail()
		})
	})

	// The reconciler verifies claims in the background (every few seconds in this
	// environment), so a claim goes from "no result" to its outcome by itself. A
	// did:key subject carries its own key, which keeps the flow free of any network.
	ginkgo.Context("verification", func() {
		const verifyTimeout, verifyPoll = 90 * time.Second, 3 * time.Second

		var (
			goodSubject, otherSubject string
			goodKeyPath               string
			verifiedCID, failedCID    string
			claimResult               = func(cid, role string) string {
				var status identityStatusOutput

				output := testEnv.CLI.Command("identity").WithArgs("status", cid, "--output", "json").ShouldSucceed()
				gomega.Expect(json.Unmarshal([]byte(output), &status)).To(gomega.Succeed())

				claim := status.Identity
				if role == "owner" {
					claim = status.Owner
				}

				if claim == nil {
					return "no result"
				}

				result, _ := claim["status"].(string)

				return result
			}
			searchCIDs = func(args ...string) string {
				return testEnv.CLI.Command("search").WithArgs(append([]string{"--format", "cid"}, args...)...).ShouldSucceed()
			}
		)

		// writeDIDKey writes a fresh Ed25519 key as PEM and returns its path and did:key.
		writeDIDKey := func(name string) (string, string) {
			pub, priv, err := ed25519.GenerateKey(rand.Reader)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			der, err := x509.MarshalPKCS8PrivateKey(priv)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			path := filepath.Join(tempDir, name+".key")
			gomega.Expect(os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600)).To(gomega.Succeed())

			encoded, err := multibase.Encode(multibase.Base58BTC, append([]byte{0xed, 0x01}, pub...))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			return path, "did:key:" + encoded
		}

		ginkgo.It("should verify a claim signed with the key its subject carries", func() {
			goodKeyPath, goodSubject = writeDIDKey("good")
			_, otherSubject = writeDIDKey("other")

			verifiedCID = pushRecord("example.com/identity-e2e/verified", map[string]string{
				corev1.AnnotationKeyIdentity: goodSubject,
				corev1.AnnotationKeyOwner:    goodSubject,
			})

			gomega.Expect(claimResult(verifiedCID, "identity")).To(gomega.Equal("no result"), "nothing is verified before a claim")

			_ = claim("--record", verifiedCID, "--role", "identity", "--key", goodKeyPath).ShouldSucceed()
			_ = claim("--record", verifiedCID, "--role", "owner", "--key", goodKeyPath).ShouldSucceed()

			gomega.Eventually(func() []string {
				return []string{claimResult(verifiedCID, "identity"), claimResult(verifiedCID, "owner")}
			}).WithTimeout(verifyTimeout).WithPolling(verifyPoll).Should(gomega.Equal([]string{"verified", "verified"}))
		})

		ginkgo.It("should print the verified status for scripts", func() {
			output := testEnv.CLI.Command("identity").WithArgs("status", verifiedCID, "--output", "raw").ShouldSucceed()

			gomega.Expect(output).To(gomega.Equal("identity verified " + goodSubject + "\nowner verified " + goodSubject))
		})

		ginkgo.It("should fail a claim signed with a key its subject does not carry", func() {
			failedCID = pushRecord("example.com/identity-e2e/failed", map[string]string{corev1.AnnotationKeyOwner: otherSubject})

			// The record declares otherSubject, but the claim is signed with the other key.
			_ = claim("--record", failedCID, "--role", "owner", "--key", goodKeyPath).ShouldSucceed()

			gomega.Eventually(func() string { return claimResult(failedCID, "owner") }).
				WithTimeout(verifyTimeout).WithPolling(verifyPoll).Should(gomega.Equal("failed"))

			output := testEnv.CLI.Command("identity").WithArgs("status", failedCID, "--output", "raw").ShouldSucceed()
			gomega.Expect(output).To(gomega.Equal("identity no-result\nowner failed " + otherSubject))
		})

		ginkgo.It("should find records by their claims", func() {
			gomega.Expect(searchCIDs("--identity-verified")).To(gomega.ContainSubstring(verifiedCID))
			gomega.Expect(searchCIDs("--owner-verified")).To(gomega.ContainSubstring(verifiedCID))
			gomega.Expect(searchCIDs("--identity", goodSubject)).To(gomega.ContainSubstring(verifiedCID))
			gomega.Expect(searchCIDs("--owner", goodSubject)).To(gomega.ContainSubstring(verifiedCID))

			// A failed claim is not a verified one.
			gomega.Expect(searchCIDs("--owner-verified")).NotTo(gomega.ContainSubstring(failedCID))
			gomega.Expect(searchCIDs("--identity-verified", "--owner", goodSubject)).NotTo(gomega.ContainSubstring(failedCID))
		})

		ginkgo.It("should only search for what is there", func() {
			_ = testEnv.CLI.Command("search").WithArgs("--exclude-owner", goodSubject).ShouldFail()
			_ = testEnv.CLI.Command("search").WithArgs("--exclude-identity", goodSubject).ShouldFail()

			// A false claim verified flag is accepted and is not a filter.
			_ = searchCIDs("--owner-verified=false", "--owner", goodSubject)
		})
	})

	ginkgo.Context("status", func() {
		ginkgo.It("should show the claims of a record by CID", func() {
			output := testEnv.CLI.Command("identity").WithArgs("status", recordCID, "--output", "json").ShouldSucceed()

			var status identityStatusOutput
			gomega.Expect(json.Unmarshal([]byte(output), &status)).To(gomega.Succeed())
			gomega.Expect(status.CID).To(gomega.Equal(recordCID))
		})

		ginkgo.It("should show the claims of a record by name", func() {
			output := testEnv.CLI.Command("identity").WithArgs("status", recordName+":v4.0.0", "--output", "json").ShouldSucceed()

			var status identityStatusOutput
			gomega.Expect(json.Unmarshal([]byte(output), &status)).To(gomega.Succeed())
			gomega.Expect(status.CID).To(gomega.Equal(recordCID))
		})

		ginkgo.It("should print a human-readable result", func() {
			output := testEnv.CLI.Command("identity").WithArgs("status", recordCID).ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
			gomega.Expect(output).To(gomega.ContainSubstring("Identity:"))
			gomega.Expect(output).To(gomega.ContainSubstring("Owner:"))
		})

		ginkgo.It("should fail for a record that does not exist", func() {
			_ = testEnv.CLI.Command("identity").WithArgs("status", "nonexistent.example.com/agent").ShouldFail()
		})
	})

	ginkgo.Context("resolve", func() {
		ginkgo.It("should resolve a name to its versions", func() {
			output := testEnv.CLI.Command("identity").WithArgs("resolve", recordName, "--output", "json").ShouldSucceed()

			var records []identityResolveOutput
			gomega.Expect(json.Unmarshal([]byte(output), &records)).To(gomega.Succeed())
			gomega.Expect(records).NotTo(gomega.BeEmpty())
			gomega.Expect(records).To(gomega.ContainElement(identityResolveOutput{Name: recordName, Version: "v4.0.0", CID: recordCID}))
		})

		ginkgo.It("should resolve a name with a version", func() {
			output := testEnv.CLI.Command("identity").WithArgs("resolve", recordName+":v4.0.0", "--output", "raw").ShouldSucceed()

			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("should fail for a version that does not exist", func() {
			_ = testEnv.CLI.Command("identity").WithArgs("resolve", recordName+":v99.0.0").ShouldFail()
		})

		ginkgo.It("should fail for a name that does not exist", func() {
			_ = testEnv.CLI.Command("identity").WithArgs("resolve", "nonexistent.example.com/agent").ShouldFail()
		})
	})
})
