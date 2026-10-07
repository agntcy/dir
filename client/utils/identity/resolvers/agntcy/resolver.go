// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agntcy

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

const (
	maxBodyBytes              = 1 << 20
	defaultRequestTimeout     = 10 * time.Second
	defaultMaxVerificationAge = 30 * time.Minute
)

// Config describes an administrator-selected authority. A nil RequireAgentBadge
// requires the badge profile by default. BearerTokenFile is optional when
// transport authentication is supplied by an operator-provided HTTP client.
type Config struct {
	VerifierURL             string        `json:"verifier_url,omitempty"               mapstructure:"verifier_url"`
	VerifierTrustBundleFile string        `json:"verifier_trust_bundle_file,omitempty" mapstructure:"verifier_trust_bundle_file"`
	VerifierID              string        `json:"verifier_id,omitempty"                mapstructure:"verifier_id"`
	Profile                 string        `json:"profile,omitempty"                    mapstructure:"profile"`
	RequireAgentBadge       *bool         `json:"require_agent_badge,omitempty"        mapstructure:"require_agent_badge"`
	BearerTokenFile         string        `json:"bearer_token_file,omitempty"          mapstructure:"bearer_token_file"`
	RequestTimeout          time.Duration `json:"request_timeout,omitempty"            mapstructure:"request_timeout"`
	MaxVerificationAge      time.Duration `json:"max_verification_age,omitempty"       mapstructure:"max_verification_age"`
}

func (c Config) RequiresBadge() bool { return c.RequireAgentBadge == nil || *c.RequireAgentBadge }

// Resolver implements key resolution and the optional record-aware contract.
// Reconciliation uses ResolveClaim once and verifies the signature locally.
// Instances are used sequentially for one reconciliation run.
type Resolver struct {
	config     Config
	publicKeys []crypto.PublicKey
	client     *http.Client
	token      string
	deadlines  map[string]time.Time
}

var (
	_ resolvers.Resolver           = (*Resolver)(nil)
	_ clientidentity.ClaimResolver = (*Resolver)(nil)
)

func (config Config) validate() error {
	u, err := url.Parse(config.VerifierURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return errors.New("AGNTCY verifier must be an administrator-configured HTTPS URL")
	}

	if config.Profile != config.claimProfile() {
		return errors.New("unsupported AGNTCY verification profile")
	}

	if config.RequestTimeout < 0 || config.MaxVerificationAge < 0 {
		return errors.New("AGNTCY timeouts and maximum age must be positive")
	}

	return nil
}

func (config Config) claimProfile() string {
	if config.RequiresBadge() {
		return Profile
	}

	return KeyProfile
}

func (config Config) withDefaults() Config {
	if config.Profile == "" {
		config.Profile = config.claimProfile()
	}
	if config.VerifierID == "" {
		config.VerifierID = "agntcy-identity-verifier"
	}

	if config.RequestTimeout == 0 {
		config.RequestTimeout = defaultRequestTimeout
	}

	if config.MaxVerificationAge == 0 {
		config.MaxVerificationAge = defaultMaxVerificationAge
	}

	return config
}

func New(config Config, client *http.Client) (*Resolver, error) {
	config = config.withDefaults()
	if err := config.validate(); err != nil {
		return nil, err
	}

	keys, err := loadPublicKeys(config.VerifierTrustBundleFile)
	if err != nil {
		return nil, err
	}

	var token string

	if config.BearerTokenFile != "" {
		data, err := os.ReadFile(config.BearerTokenFile)
		if err != nil {
			return nil, fmt.Errorf("read verifier authentication token: %w", err)
		}

		token = strings.TrimSpace(string(data))
		if token == "" || strings.ContainsAny(token, "\r\n") {
			return nil, errors.New("invalid verifier authentication token")
		}
	}

	if client == nil {
		client = &http.Client{}
	}
	// Copy the supplied client: caller-owned TLS settings survive, while every
	// request is bounded and redirects cannot forward credentials elsewhere.
	bounded := *client
	bounded.Timeout = config.RequestTimeout
	bounded.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }

	return &Resolver{config: config, publicKeys: keys, client: &bounded, token: token, deadlines: map[string]time.Time{}}, nil
}

func (r *Resolver) Resolve(ctx context.Context, subject string, certificate []byte) ([]crypto.PublicKey, error) {
	if _, err := AgentID(subject); err != nil {
		return nil, err
	}

	if len(certificate) != 0 {
		return nil, errors.New("AGNTCY claims do not carry X.509 certificates")
	}

	nonce, err := newNonce()
	if err != nil {
		return nil, err
	}

	request := ResolutionRequest{Subject: subject, Nonce: nonce}

	result, until, err := r.call(ctx, "/v1/resolve", request, subject, "", "resolve", KeyProfile)
	if err != nil {
		return nil, err
	}

	keys, err := resultKeys(result)
	if err != nil {
		return nil, err
	}

	r.deadlines[subject] = until

	return keys, nil
}

func resultKeys(result VerificationResult) ([]crypto.PublicKey, error) {
	keys := make([]crypto.PublicKey, 0, len(result.PublicKeys))
	for _, raw := range result.PublicKeys {
		key, err := jwk.ParseKey(raw)
		if err != nil {
			continue
		}
		if pub, ok := jws.PublicKeyFromJWK(key); ok {
			keys = append(keys, pub)
		}
	}
	if len(keys) == 0 {
		return nil, resolvers.ErrNoKeys
	}
	return keys, nil
}

// ResolutionValidUntil bounds cached keys, including when the badge profile is
// disabled. The reconciler persists this deadline with the accepted claim.
func (r *Resolver) ResolutionValidUntil(subject string, _ []byte) time.Time {
	return r.deadlines[subject]
}

// ResolveClaim obtains keys and evidence for one exact signed claim. No
// preliminary Resolve call or subject-level evidence cache is used.
func (r *Resolver) ResolveClaim(ctx context.Context, claim *identityv1.Claim) (clientidentity.ClaimResolution, error) {
	empty := clientidentity.ClaimResolution{}
	if claim == nil {
		return empty, errors.New("claim is nil")
	}
	if _, err := AgentID(claim.GetSubject()); err != nil {
		return empty, err
	}

	if err := clientidentity.Check(claim, claim.GetRecordCid(), claim.GetSubject()); err != nil {
		return empty, fmt.Errorf("check AGNTCY claim: %w", err)
	}
	payload, err := claim.GetPayload()
	if err != nil {
		return empty, fmt.Errorf("canonical claim payload: %w", err)
	}
	nonce, err := newNonce()
	if err != nil {
		return empty, err
	}

	request := VerificationRequest{Profile: r.config.claimProfile(), Subject: claim.GetSubject(), Signature: claim.GetSignature(), Payload: base64.RawURLEncoding.EncodeToString(payload), Nonce: nonce}

	result, until, err := r.call(ctx, "/v1/verify", request, claim.GetSubject(), claim.GetRecordCid(), "verify", request.Profile)
	if err != nil {
		return empty, err
	}

	if !result.Checks.Identity || r.config.RequiresBadge() && !result.Checks.Badge {
		return empty, errors.New("required AGNTCY verification check failed")
	}

	keys, err := resultKeys(result)
	if err != nil {
		return empty, err
	}

	return clientidentity.ClaimResolution{PublicKeys: keys, ValidUntil: until}, nil
}

func (r *Resolver) call(ctx context.Context, path string, input any, subject, cid, kind, profile string) (VerificationResult, time.Time, error) {
	var result VerificationResult

	data, err := json.Marshal(input)
	if err != nil {
		return result, time.Time{}, fmt.Errorf("encode verifier request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(r.config.VerifierURL, "/")+path, bytes.NewReader(data))
	if err != nil {
		return result, time.Time{}, fmt.Errorf("construct verifier request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if r.token != "" {
		req.Header.Set("Authorization", "Bearer "+r.token)
	}

	response, err := r.client.Do(req)
	if err != nil {
		return result, time.Time{}, fmt.Errorf("call AGNTCY verifier: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		return result, time.Time{}, &safefetch.StatusError{Code: response.StatusCode}
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes+1))
	if err != nil {
		return result, time.Time{}, fmt.Errorf("read verifier response: %w", err)
	}

	if len(body) > maxBodyBytes {
		return result, time.Time{}, errors.New("verifier response exceeds size limit")
	}

	var envelope VerificationResponse
	if err := json.Unmarshal(body, &envelope); err != nil {
		return result, time.Time{}, errors.New("invalid verifier response")
	}

	parts := strings.Split(envelope.ResultJWS, ".")
	if len(parts) != 3 || parts[1] == "" {
		return result, time.Time{}, errors.New("expected a signed embedded-payload verifier result")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return result, time.Time{}, errors.New("invalid verifier result payload")
	}

	if err := jws.Verify(parts[0]+".."+parts[2], payload, r.publicKeys...); err != nil {
		return result, time.Time{}, fmt.Errorf("verify authority signature: %w", err)
	}

	if err := json.Unmarshal(payload, &result); err != nil {
		return result, time.Time{}, errors.New("invalid signed verifier result")
	}

	until, err := r.validate(result, DigestRequest(data), subject, cid, kind, profile, time.Now())

	return result, until, err
}

func (r *Resolver) validate(result VerificationResult, digest, subject, cid, kind, profile string, now time.Time) (time.Time, error) {
	if result.Version != ProtocolVersion || result.Kind != kind || result.Profile != profile || result.PolicyVersion != SubjectKeyPolicy || result.Verifier != r.config.VerifierID {
		return time.Time{}, errors.New("unexpected verifier identity, protocol, profile or policy")
	}

	if result.Subject != subject || result.RecordCID != cid || result.RequestDigest != digest {
		return time.Time{}, errors.New("verifier result is not bound to the complete requested claim")
	}

	if !result.Verified {
		return time.Time{}, errors.New("AGNTCY verifier rejected the identity evidence")
	}

	return r.validateWindow(result, now)
}

func (r *Resolver) validateWindow(result VerificationResult, now time.Time) (time.Time, error) {
	checked, err := time.Parse(time.RFC3339, result.CheckedAt)
	if err != nil || checked.After(now.Add(time.Minute)) || now.Sub(checked) > r.config.MaxVerificationAge {
		return time.Time{}, errors.New("verifier check time is invalid or stale")
	}

	expires, err := time.Parse(time.RFC3339, result.ExpiresAt)
	if err != nil || !expires.After(now) || !expires.After(checked) {
		return time.Time{}, errors.New("verifier result is expired or has an invalid validity window")
	}

	if deadline := checked.Add(r.config.MaxVerificationAge); deadline.Before(expires) {
		expires = deadline
	}

	if !expires.After(now) {
		return time.Time{}, errors.New("verifier result exceeds maximum verification age")
	}

	return expires, nil
}

func loadPublicKeys(path string) ([]crypto.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read verifier trust bundle: %w", err)
	}

	if block, _ := pem.Decode(data); block != nil {
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse verifier public key: %w", err)
		}

		key, ok := jws.ToPublicKey(pub)
		if !ok {
			return nil, resolvers.ErrNoKeys
		}

		return []crypto.PublicKey{key}, nil
	}

	set, err := jwk.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse verifier JWKS: %w", err)
	}

	var keys []crypto.PublicKey

	for i := range set.Len() {
		key, _ := set.Key(i)
		if pub, ok := jws.PublicKeyFromJWK(key); ok {
			keys = append(keys, pub)
		}
	}

	if len(keys) == 0 {
		return nil, resolvers.ErrNoKeys
	}

	return keys, nil
}

func newNonce() (string, error) {
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", fmt.Errorf("generate request nonce: %w", err)
	}

	return hex.EncodeToString(nonce[:]), nil
}
