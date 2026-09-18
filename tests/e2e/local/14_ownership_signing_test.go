// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"os"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/tests/e2e/shared/testdata"
	"github.com/agntcy/dir/tests/e2e/shared/utils"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// Tests for SPIFFE-based signing of ownership claims.
//
// Security model:
//   - A signed claim carries an ECDSA signature over SHA-256(owner_id + ":" + claimed_at)
//     and the DER-encoded X.509 leaf certificate from the signer's SPIFFE SVID.
//   - The CLI verifies at push time that the certificate's URI SAN matches owner_id.
//   - The reconciler verifies the signature before indexing; claims with an invalid
//     signature are rejected regardless of how they reached the OCI store.

const (
	testSignedOwnerID    = "spiffe://test.example.org/agent/signed-owner"
	testMismatchedCertID = "spiffe://test.example.org/agent/different-identity"
	// testMismatchOwnerID is the claimed owner in the identity-mismatch context.
	// It must differ from testSignedOwnerID so the "not indexed" assertion doesn't
	// collide with the happy-path indexing (both contexts push the same JSON → same CID).
	testMismatchOwnerID = "spiffe://test.example.org/agent/mismatch-owner"
)

var _ = ginkgo.Describe("Ownership claim signing", func() {
	ginkgo.BeforeEach(func() {
		utils.ResetCLIState()
	})

	// -------------------------------------------------------------------------
	// Context 1: happy path — signed claim is accepted and becomes searchable
	// -------------------------------------------------------------------------
	ginkgo.Context("signed claim is indexed and searchable", ginkgo.Ordered, func() {
		var (
			tempDir   string
			recordCID string
			keyPath   string
			certPath  string
		)

		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "ownership-signing-happy-*")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath := filepath.Join(tempDir, "record.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())

			keyPath, certPath = utils.GenerateSpiffeTestCert(tempDir, testSignedOwnerID)
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("pushes a signed ownership claim without error", func() {
			testEnv.CLI.Ownership().
				ClaimSigned(recordCID, testSignedOwnerID, keyPath, certPath).
				ShouldSucceed()
		})

		ginkgo.It("finds the record by exact owner after the reconciler indexes the signed claim", func() {
			// Allow the reconciler's ownership task to walk the referrer and index it.
			time.Sleep(10 * time.Second)

			output := testEnv.CLI.Search().
				WithOwner(testSignedOwnerID).
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})

		ginkgo.It("finds the record with a wildcard over the SPIFFE domain", func() {
			output := testEnv.CLI.Search().
				WithOwner("spiffe://test.example.org/*").
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})
	})

	// -------------------------------------------------------------------------
	// Context 2: identity mismatch — CLI rejects before pushing
	// -------------------------------------------------------------------------
	ginkgo.Context("cert SPIFFE ID does not match owner_id", ginkgo.Ordered, func() {
		var (
			tempDir   string
			recordCID string
			keyPath   string
			certPath  string
		)

		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "ownership-signing-mismatch-*")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath := filepath.Join(tempDir, "record.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())

			// Cert identifies as testMismatchedCertID, but we will pass testMismatchOwnerID
			// as --owner. These differ, so the CLI must reject the claim before pushing.
			keyPath, certPath = utils.GenerateSpiffeTestCert(tempDir, testMismatchedCertID)
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("fails when the certificate SPIFFE ID does not match the claimed owner_id", func() {
			// cert URI SAN = testMismatchedCertID, owner = testMismatchOwnerID → must fail
			_ = testEnv.CLI.Ownership().
				ClaimSigned(recordCID, testMismatchOwnerID, keyPath, certPath).
				ShouldFail()
		})

		ginkgo.It("does not index the rejected claim", func() {
			// testMismatchOwnerID was never successfully pushed to OCI, so no referrer
			// exists and the reconciler will never index it. Searching by this owner must
			// return no results — in particular, recordCID must not appear.
			output := testEnv.CLI.Search().
				WithOwner(testMismatchOwnerID).
				ShouldSucceed()
			gomega.Expect(output).NotTo(gomega.ContainSubstring(recordCID))
		})
	})

	// -------------------------------------------------------------------------
	// Context 3: re-push of an identical signed claim is idempotent
	// -------------------------------------------------------------------------
	// Tamper-detection (corrupted signature bytes, altered owner_id after signing)
	// is covered by unit tests in api/ownership/v1/sign_test.go.
	ginkgo.Context("re-pushing an identical signed claim is idempotent", ginkgo.Ordered, func() {
		var (
			tempDir   string
			recordCID string
			keyPath   string
			certPath  string
		)

		ginkgo.BeforeAll(func() {
			var err error

			tempDir, err = os.MkdirTemp("", "ownership-signing-idempotent-*")
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordPath := filepath.Join(tempDir, "record.json")
			err = os.WriteFile(recordPath, testdata.ExpectedRecordV080V4JSON, 0o600)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			recordCID = testEnv.CLI.Push(recordPath).WithArgs("--output", "raw").ShouldSucceed()
			gomega.Expect(recordCID).NotTo(gomega.BeEmpty())

			keyPath, certPath = utils.GenerateSpiffeTestCert(tempDir, testSignedOwnerID)
		})

		ginkgo.AfterAll(func() {
			if tempDir != "" {
				_ = os.RemoveAll(tempDir)
			}
		})

		ginkgo.It("accepts a second push of the same signed claim without error", func() {
			testEnv.CLI.Ownership().
				ClaimSigned(recordCID, testSignedOwnerID, keyPath, certPath).
				ShouldSucceed()

			// Push the same claim again — idempotent, must not error.
			testEnv.CLI.Ownership().
				ClaimSigned(recordCID, testSignedOwnerID, keyPath, certPath).
				ShouldSucceed()
		})

		ginkgo.It("record remains searchable by owner after the second push", func() {
			time.Sleep(10 * time.Second)

			output := testEnv.CLI.Search().
				WithOwner(testSignedOwnerID).
				ShouldSucceed()
			gomega.Expect(output).To(gomega.ContainSubstring(recordCID))
		})
	})
})
