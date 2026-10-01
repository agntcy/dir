// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/youmark/pkcs8"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const testCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"

// ---- fakes: just enough of the server for the commands ----

type pushStream struct {
	grpc.ClientStream
	sent []*storev1.PushReferrerRequest
}

func (s *pushStream) Send(r *storev1.PushReferrerRequest) error {
	s.sent = append(s.sent, r)

	return nil
}

func (s *pushStream) CloseSend() error { return nil }

func (s *pushStream) Recv() (*storev1.PushReferrerResponse, error) {
	return &storev1.PushReferrerResponse{Success: true}, nil
}

type fakeStore struct {
	storev1.StoreServiceClient
	stream *pushStream
}

func (f *fakeStore) PushReferrer(context.Context, ...grpc.CallOption) (storev1.StoreService_PushReferrerClient, error) {
	return f.stream, nil
}

type fakeIdentity struct {
	identityv1.IdentityServiceClient
	status  *identityv1.GetIdentityStatusResponse
	records []*corev1.NamedRecordRef
	gotCID  string
	gotName string
	gotVer  string
}

func (f *fakeIdentity) GetIdentityStatus(_ context.Context, req *identityv1.GetIdentityStatusRequest, _ ...grpc.CallOption) (*identityv1.GetIdentityStatusResponse, error) {
	f.gotCID = req.GetCid()

	return f.status, nil
}

func (f *fakeIdentity) Resolve(_ context.Context, req *identityv1.ResolveRequest, _ ...grpc.CallOption) (*identityv1.ResolveResponse, error) {
	f.gotName, f.gotVer = req.GetName(), req.GetVersion()

	return &identityv1.ResolveResponse{Records: f.records}, nil
}

// newCommand returns a command whose context carries c, with the output flag
// the real commands have, writing to the returned buffer.
func newCommand(t *testing.T, c *client.Client, args ...string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()

	cmd := &cobra.Command{Use: "test"}
	presenter.AddOutputFlags(cmd)
	require.NoError(t, cmd.ParseFlags(args))

	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetContext(ctxUtils.SetClientForContext(context.Background(), c))

	return cmd, out
}

// ---- key material ----

func writeKey(t *testing.T, key *ecdsa.PrivateKey, password string) string {
	t.Helper()

	var block *pem.Block

	if password == "" {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, err)

		block = &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	} else {
		der, err := pkcs8.MarshalPrivateKey(key, []byte(password), pkcs8.DefaultOpts)
		require.NoError(t, err)

		block = &pem.Block{Type: encryptedKeyPEMType, Bytes: der}
	}

	path := filepath.Join(t.TempDir(), "key.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(block), 0o600))

	return path
}

func writeCert(t *testing.T, key *ecdsa.PrivateKey, subject string) string {
	t.Helper()

	uri, err := url.Parse(subject)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: subject},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "cert.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))

	return path
}

func newKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	return key
}

func setClaimOpts(t *testing.T, role, subject, key, cert string) {
	t.Helper()

	claimOpts.Record, claimOpts.Role, claimOpts.Subject = testCID, role, subject
	claimOpts.Key, claimOpts.Cert, claimOpts.PasswordStdin = key, cert, false

	t.Cleanup(func() {
		claimOpts = struct {
			Record        string
			Role          string
			Subject       string
			Key           string
			PasswordStdin bool
			Cert          string
		}{}
	})
}

// ---- claim ----

func runClaimWith(t *testing.T) (*pushStream, *bytes.Buffer, error) {
	t.Helper()

	stream := &pushStream{}
	cmd, out := newCommand(t, &client.Client{StoreServiceClient: &fakeStore{stream: stream}})

	return stream, out, runClaim(cmd)
}

func pushedClaim(t *testing.T, stream *pushStream) (*storev1.PushReferrerRequest, *identityv1.Claim) {
	t.Helper()

	require.Len(t, stream.sent, 1)

	claim := &identityv1.Claim{}
	require.NoError(t, claim.UnmarshalReferrer(&corev1.RecordReferrer{Type: stream.sent[0].GetType(), Data: stream.sent[0].GetData()}))

	return stream.sent[0], claim
}

// Subjects that are not SPIFFE need only --key: no --cert.
func TestClaim_KeyOnlyForNonSPIFFESubjects(t *testing.T) {
	for _, subject := range []string{"dns:acme.com", "acme.com", "https://acme.com/agents", "did:web:acme.com", "did:key:z6Mk"} {
		t.Run(subject, func(t *testing.T) {
			key := newKey(t)
			setClaimOpts(t, roleIdentity, subject, writeKey(t, key, ""), "")

			stream, _, err := runClaimWith(t)
			require.NoError(t, err)

			req, claim := pushedClaim(t, stream)
			assert.Equal(t, testCID, req.GetRecordRef().GetCid())
			assert.Equal(t, corev1.IdentityClaimReferrerType, req.GetType())
			assert.Nil(t, claim.Certificate)

			ok, err := clientidentity.Verify(claim, testCID, subject, &key.PublicKey)
			require.NoError(t, err)
			assert.True(t, ok)
		})
	}
}

func TestClaim_OwnerRole(t *testing.T) {
	key := newKey(t)
	setClaimOpts(t, roleOwner, "dns:acme.com", writeKey(t, key, ""), "")

	stream, out, err := runClaimWith(t)
	require.NoError(t, err)

	req, claim := pushedClaim(t, stream)
	assert.Equal(t, corev1.OwnershipClaimReferrerType, req.GetType())
	assert.Equal(t, identityv1.ClaimRole_CLAIM_ROLE_OWNER, claim.GetRole())
	assert.Contains(t, out.String(), "Pushed owner claim dns:acme.com for "+testCID)
	assert.Contains(t, out.String(), "dirctl identity status "+testCID)
}

func TestClaim_SPIFFEWithCertificate(t *testing.T) {
	const subject = "spiffe://acme.com/agents/finance"

	key := newKey(t)
	setClaimOpts(t, roleIdentity, subject, writeKey(t, key, ""), writeCert(t, key, subject))

	stream, _, err := runClaimWith(t)
	require.NoError(t, err)

	_, claim := pushedClaim(t, stream)
	assert.NotEmpty(t, claim.GetCertificate())
}

func TestClaim_EncryptedKey(t *testing.T) {
	key := newKey(t)
	setClaimOpts(t, roleIdentity, "dns:acme.com", writeKey(t, key, "s3cret"), "")
	t.Setenv("COSIGN_PASSWORD", "s3cret")

	stream, _, err := runClaimWith(t)
	require.NoError(t, err)

	_, claim := pushedClaim(t, stream)

	ok, err := clientidentity.Verify(claim, testCID, "dns:acme.com", &key.PublicKey)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestClaim_Rejects(t *testing.T) {
	const spiffeSubject = "spiffe://acme.com/agents/finance"

	key := newKey(t)
	keyPath := writeKey(t, key, "")
	certPath := writeCert(t, key, spiffeSubject)

	tests := map[string]struct {
		role, subject, key, cert, wantErr string
	}{
		"unknown role":                 {"admin", "dns:acme.com", keyPath, "", "invalid --role"},
		"cert on a non-spiffe subject": {roleIdentity, "dns:acme.com", keyPath, certPath, "only applies to spiffe://"},
		"spiffe subject without cert":  {roleIdentity, spiffeSubject, keyPath, "", "--cert is required"},
		"missing key file":             {roleIdentity, "dns:acme.com", filepath.Join(t.TempDir(), "nope"), "", "failed to read private key"},
		"missing cert file":            {roleIdentity, spiffeSubject, keyPath, filepath.Join(t.TempDir(), "nope"), "failed to read certificate"},
		"cert of another subject":      {roleIdentity, "spiffe://acme.com/agents/other", keyPath, certPath, "does not cover"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			setClaimOpts(t, tt.role, tt.subject, tt.key, tt.cert)

			stream, _, err := runClaimWith(t)
			require.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, stream.sent, "nothing is pushed for a rejected claim")
		})
	}
}

func TestClaim_EncryptedKeyNeedsPassword(t *testing.T) {
	key := newKey(t)
	setClaimOpts(t, roleIdentity, "dns:acme.com", writeKey(t, key, "s3cret"), "")

	t.Run("no password source", func(t *testing.T) {
		_, err := loadSigner(claimOpts.Key, false)
		require.Error(t, err)
	})

	t.Run("wrong password", func(t *testing.T) {
		t.Setenv("COSIGN_PASSWORD", "wrong")

		_, err := loadSigner(claimOpts.Key, false)
		require.ErrorContains(t, err, "failed to load private key")
	})
}

func TestLoadSigner_UnencryptedKeyNeverAsksForPassword(t *testing.T) {
	// Neither COSIGN_PASSWORD nor --password-stdin is set, and stdin is not
	// read: an unencrypted key must load without any of them.
	_, err := loadSigner(writeKey(t, newKey(t), ""), false)
	require.NoError(t, err)
}

func TestReadPassword(t *testing.T) {
	noTerminal := func() bool { return false }

	t.Run("environment first", func(t *testing.T) {
		pw, err := readPassword(passwordSource{
			lookupEnv:     func() (string, bool) { return "from-env", true },
			passwordStdin: true,
			stdin:         strings.NewReader("from-stdin"),
			isTerminal:    noTerminal,
		})
		require.NoError(t, err)
		assert.Equal(t, "from-env", string(pw))
	})

	t.Run("stdin when asked, without its line break", func(t *testing.T) {
		for _, in := range []string{"from-stdin", "from-stdin\n", "from-stdin\r\n"} {
			pw, err := readPassword(passwordSource{
				lookupEnv:     func() (string, bool) { return "", false },
				passwordStdin: true,
				stdin:         strings.NewReader(in),
				isTerminal:    noTerminal,
			})
			require.NoError(t, err)
			assert.Equal(t, "from-stdin", string(pw), "%q", in)
		}
	})

	t.Run("terminal prompt", func(t *testing.T) {
		pw, err := readPassword(passwordSource{
			lookupEnv:    func() (string, bool) { return "", false },
			isTerminal:   func() bool { return true },
			readTerminal: func(bool) ([]byte, error) { return []byte("typed"), nil },
		})
		require.NoError(t, err)
		assert.Equal(t, "typed", string(pw))
	})

	t.Run("nowhere to read it from", func(t *testing.T) {
		_, err := readPassword(passwordSource{
			lookupEnv:  func() (string, bool) { return "", false },
			isTerminal: noTerminal,
		})
		require.ErrorContains(t, err, "COSIGN_PASSWORD")
	})

	t.Run("terminal failure", func(t *testing.T) {
		_, err := readPassword(passwordSource{
			lookupEnv:    func() (string, bool) { return "", false },
			isTerminal:   func() bool { return true },
			readTerminal: func(bool) ([]byte, error) { return nil, errors.New("no tty") },
		})
		require.ErrorContains(t, err, "no tty")
		require.ErrorContains(t, err, "COSIGN_PASSWORD")
	})
}

// ---- status ----

func TestStatus(t *testing.T) {
	verifiedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	failure := "no key found"
	fake := &fakeIdentity{status: &identityv1.GetIdentityStatusResponse{
		Identity: &identityv1.ClaimVerification{
			Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: "did:web:acme.com:agent",
			Status:     identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED,
			VerifiedAt: timestamppb.New(verifiedAt),
		},
		Owner: &identityv1.ClaimVerification{
			Role: identityv1.ClaimRole_CLAIM_ROLE_OWNER, Subject: "dns:acme.com",
			Status: identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_FAILED, Error: &failure,
			VerifiedAt: timestamppb.New(verifiedAt),
		},
	}}

	t.Run("human", func(t *testing.T) {
		cmd, out := newCommand(t, &client.Client{IdentityServiceClient: fake})
		require.NoError(t, runStatus(cmd, testCID))

		assert.Equal(t, testCID, fake.gotCID)
		assert.Contains(t, out.String(), "verified did:web:acme.com:agent")
		assert.Contains(t, out.String(), "failed dns:acme.com")
		assert.Contains(t, out.String(), "no key found")
	})

	t.Run("json", func(t *testing.T) {
		cmd, out := newCommand(t, &client.Client{IdentityServiceClient: fake}, "--output", "json")
		require.NoError(t, runStatus(cmd, testCID))

		var got statusResult
		require.NoError(t, json.Unmarshal(out.Bytes(), &got))
		assert.Equal(t, testCID, got.CID)
		assert.Equal(t, "identity", got.Identity.Role)
		assert.Equal(t, "verified", got.Identity.Status)
		assert.Equal(t, "2026-01-02T03:04:05Z", got.Identity.VerifiedAt)
		assert.Equal(t, "failed", got.Owner.Status)
		assert.Equal(t, "no key found", got.Owner.Error)
	})
}

func TestStatus_NoClaimIsNotAnError(t *testing.T) {
	fake := &fakeIdentity{status: &identityv1.GetIdentityStatusResponse{}}

	cmd, out := newCommand(t, &client.Client{IdentityServiceClient: fake})
	require.NoError(t, runStatus(cmd, testCID))
	assert.Equal(t, 2, strings.Count(out.String(), "no result"))

	cmd, out = newCommand(t, &client.Client{IdentityServiceClient: fake}, "--output", "json")
	require.NoError(t, runStatus(cmd, testCID))

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Contains(t, got, "identity")
	assert.Nil(t, got["identity"])
	assert.Nil(t, got["owner"])
}

// ---- resolve ----

func TestResolve(t *testing.T) {
	fake := &fakeIdentity{records: []*corev1.NamedRecordRef{
		{Name: "acme.com/agent", Version: "2.0.0", Cid: "cid-v2"},
		{Name: "acme.com/agent", Version: "1.0.0", Cid: "cid-v1"},
	}}
	c := &client.Client{IdentityServiceClient: fake}

	t.Run("all versions", func(t *testing.T) {
		cmd, out := newCommand(t, c)
		require.NoError(t, runResolve(cmd, "acme.com/agent"))

		assert.Equal(t, "acme.com/agent", fake.gotName)
		assert.Empty(t, fake.gotVer)
		assert.Equal(t, "cid-v2  acme.com/agent  2.0.0\ncid-v1  acme.com/agent  1.0.0\n", out.String())
	})

	t.Run("one version", func(t *testing.T) {
		cmd, _ := newCommand(t, c)
		require.NoError(t, runResolve(cmd, "acme.com/agent:1.0.0"))

		assert.Equal(t, "acme.com/agent", fake.gotName)
		assert.Equal(t, "1.0.0", fake.gotVer)
	})

	t.Run("json", func(t *testing.T) {
		cmd, out := newCommand(t, c, "--output", "json")
		require.NoError(t, runResolve(cmd, "acme.com/agent"))

		var got []resolvedRecord
		require.NoError(t, json.Unmarshal(out.Bytes(), &got))
		assert.Equal(t, []resolvedRecord{
			{Name: "acme.com/agent", Version: "2.0.0", CID: "cid-v2"},
			{Name: "acme.com/agent", Version: "1.0.0", CID: "cid-v1"},
		}, got)
	})

	t.Run("raw prints only the CIDs", func(t *testing.T) {
		cmd, out := newCommand(t, c, "--output", "raw")
		require.NoError(t, runResolve(cmd, "acme.com/agent"))
		assert.Equal(t, "cid-v2\ncid-v1\n", out.String())
	})

	t.Run("a CID is not a name", func(t *testing.T) {
		cmd, _ := newCommand(t, c)
		require.ErrorContains(t, runResolve(cmd, testCID), "is a CID")
	})
}
