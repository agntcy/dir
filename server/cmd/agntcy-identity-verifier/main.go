// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// agntcy-identity-verifier externalizes AGNTCY-specific identity resolution,
// proof-of-control verification, Agent Badge validation, and CID binding.
package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	clientidentity "github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/identity/resolvers/agntcy"
	clientjws "github.com/agntcy/dir/client/utils/jws"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	maxBodyBytes        = 4 << 20
	resultTTL           = 30 * time.Minute
	nodeTimeout         = 10 * time.Second
	headerTimeout       = 5 * time.Second
	readTimeout         = 15 * time.Second
	writeTimeout        = 30 * time.Second
	verificationTimeout = 20 * time.Second
	compactSegments     = 3
	minimumRSABits      = 2048
)

type server struct {
	identityURL string
	verifierID  string
	privateKey  *rsa.PrivateKey
	client      *http.Client
	token       string
	ttl         time.Duration
}

type resolveResponse struct {
	ResolverMetadata struct {
		ID                 string `json:"id"`
		VerificationMethod []struct {
			PublicKeyJWK json.RawMessage `json:"publicKeyJwk"`
		} `json:"verificationMethod"`
	} `json:"resolverMetadata"`
}

type credentialEnvelope struct {
	EnvelopeType string `json:"envelopeType"`
	Value        string `json:"value"`
}

type wellKnownResponse struct {
	VCs []credentialEnvelope `json:"vcs"`
}

type verifyResponse struct {
	Status bool `json:"status"`
}

func main() {
	listen := envOr("LISTEN_ADDRESS", ":8443")
	identityURL := strings.TrimRight(os.Getenv("IDENTITY_NODE_URL"), "/")

	if err := validateIdentityURL(identityURL); err != nil {
		log.Fatal(err)
	}

	privateKey, err := loadPrivateKey(os.Getenv("RESULT_SIGNING_KEY_PATH"))
	if err != nil {
		log.Fatalf("load result-signing key: %v", err)
	}

	tokenBytes, err := os.ReadFile(os.Getenv("VERIFIER_BEARER_TOKEN_FILE")) //nolint:gosec // Administrator-provided secret file; no request data enters the path.
	if err != nil || strings.TrimSpace(string(tokenBytes)) == "" || strings.ContainsAny(strings.TrimSpace(string(tokenBytes)), "\r\n") {
		log.Fatal("VERIFIER_BEARER_TOKEN_FILE must contain an authentication token")
	}

	ttl, err := time.ParseDuration(envOr("RESULT_TTL", resultTTL.String()))
	if err != nil || ttl <= 0 || ttl > time.Hour {
		log.Fatal("RESULT_TTL must be positive and at most one hour")
	}

	s := &server{
		identityURL: identityURL, verifierID: envOr("VERIFIER_ID", "agntcy-identity-verifier"),
		privateKey: privateKey, token: strings.TrimSpace(string(tokenBytes)), ttl: ttl,
		client: &http.Client{Timeout: nodeTimeout, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }},
	}

	cert, key := os.Getenv("TLS_CERT_FILE"), os.Getenv("TLS_KEY_FILE")
	if cert == "" || key == "" {
		log.Fatal("TLS_CERT_FILE and TLS_KEY_FILE are required")
	}

	httpServer := &http.Server{Addr: listen, Handler: s.handler(), ReadHeaderTimeout: headerTimeout, ReadTimeout: readTimeout, WriteTimeout: writeTimeout, IdleTimeout: time.Minute}
	log.Printf("AGNTCY identity verifier listening on %s", listen)

	if err := httpServer.ListenAndServeTLS(cert, key); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func validateIdentityURL(identityURL string) error {
	u, err := url.Parse(identityURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return errors.New("IDENTITY_NODE_URL must be an administrator-configured HTTPS URL")
	}

	return nil
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"status": "ok"})
	})
	mux.HandleFunc("POST /v1/resolve", s.authorize(s.resolve))
	mux.HandleFunc("POST /v1/verify", s.authorize(s.verify))

	return mux
}

func (s *server) authorize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.token == "" || subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		next(w, r)
	}
}

func decodeRequest(w http.ResponseWriter, req *http.Request, output any) error {
	req.Body = http.MaxBytesReader(w, req.Body, maxBodyBytes)
	defer req.Body.Close() //nolint:errcheck

	decoder := json.NewDecoder(req.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(output); err != nil {
		return fmt.Errorf("decode verifier request: %w", err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing request data")
	}

	return nil
}

func (s *server) result(kind, profile, subject, cid string, request any) (agntcy.VerificationResult, error) {
	data, err := json.Marshal(request)
	if err != nil {
		return agntcy.VerificationResult{}, fmt.Errorf("encode verification request: %w", err)
	}

	now := time.Now().UTC()

	ttl := s.ttl
	if ttl == 0 {
		ttl = resultTTL
	}

	return agntcy.VerificationResult{
		Version: agntcy.ProtocolVersion, Kind: kind, Verifier: s.verifierID, Profile: profile,
		PolicyVersion: agntcy.SubjectKeyPolicy, Subject: subject, RecordCID: cid,
		RequestDigest: agntcy.DigestRequest(data), CheckedAt: now.Format(time.RFC3339), ExpiresAt: now.Add(ttl).Format(time.RFC3339),
	}, nil
}

func (s *server) respond(w http.ResponseWriter, result agntcy.VerificationResult) {
	signed, err := s.signResult(result)
	if err != nil {
		http.Error(w, "could not sign verification result", http.StatusInternalServerError)

		return
	}

	writeJSON(w, agntcy.VerificationResponse{ResultJWS: signed})
}

func (s *server) resolve(w http.ResponseWriter, req *http.Request) {
	var input agntcy.ResolutionRequest
	if err := decodeRequest(w, req, &input); err != nil || len(input.Nonce) != 64 {
		http.Error(w, "invalid resolution request", http.StatusBadRequest)

		return
	}

	id, err := agntcy.AgentID(input.Subject)
	if err != nil {
		http.Error(w, "invalid AGNTCY subject", http.StatusBadRequest)

		return
	}

	result, err := s.result("resolve", agntcy.KeyProfile, input.Subject, "", input)
	if err != nil {
		http.Error(w, "invalid verification request", http.StatusBadRequest)

		return
	}

	keys, err := s.resolveKeys(req.Context(), id)
	if err != nil {
		s.nodeFailure(w, result, err)

		return
	}

	for _, key := range keys {
		public, err := jwk.FromRaw(key)
		if err != nil {
			continue
		}

		data, err := json.Marshal(public)
		if err == nil {
			result.PublicKeys = append(result.PublicKeys, data)
		}
	}

	result.Verified = len(result.PublicKeys) > 0
	s.respond(w, result)
}

func (s *server) verify(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), verificationTimeout)
	defer cancel()

	var input agntcy.VerificationRequest
	if err := decodeRequest(w, req, &input); err != nil || len(input.Nonce) != 64 {
		http.Error(w, "invalid verification request", http.StatusBadRequest)

		return
	}

	payload, err := base64.RawURLEncoding.DecodeString(input.Payload)
	if err != nil {
		http.Error(w, "invalid canonical payload", http.StatusBadRequest)

		return
	}

	claim, err := parseCanonicalPayload(payload, input.Signature)
	if err != nil || claim.GetSubject() != input.Subject {
		http.Error(w, "canonical payload does not match request", http.StatusBadRequest)

		return
	}

	id, err := agntcy.AgentID(input.Subject)
	if err != nil {
		http.Error(w, "invalid AGNTCY subject", http.StatusBadRequest)

		return
	}

	result, err := s.result("verify", agntcy.Profile, input.Subject, claim.GetRecordCid(), input)
	if err != nil {
		http.Error(w, "invalid verification request", http.StatusBadRequest)

		return
	}

	keys, err := s.resolveKeys(ctx, id)
	if err != nil {
		s.nodeFailure(w, result, err)

		return
	}

	if _, err := clientidentity.Verify(claim, claim.GetRecordCid(), input.Subject, keys...); err != nil {
		result.Error = "proof of control failed"
		s.respond(w, result)

		return
	}

	matched, until, err := s.verifyMatchingAgentBadge(ctx, id, claim.GetRecordCid())
	if err != nil {
		s.nodeFailure(w, result, err)

		return
	}

	result.Verified = matched
	if !matched {
		result.Error = "no valid Agent Badge binds the subject to the record"
	}

	deadline, _ := time.Parse(time.RFC3339, result.ExpiresAt)
	if !until.IsZero() && until.Before(deadline) {
		result.ExpiresAt = until.UTC().Format(time.RFC3339)
	}

	s.respond(w, result)
}

// Only transport failures preserve a previous result. Authoritative missing
// keys, rejected credentials and malformed evidence produce a signed rejection.
func (s *server) nodeFailure(w http.ResponseWriter, result agntcy.VerificationResult, err error) {
	var (
		netErr    net.Error
		statusErr *safefetch.StatusError
	)
	if errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) || errors.As(err, &statusErr) && (statusErr.Code >= 500 || statusErr.Code == 429 || statusErr.Code == 408) {
		http.Error(w, "Identity Node temporarily unavailable", http.StatusServiceUnavailable)

		return
	}

	result.Verified = false
	result.Error = "Identity Node rejected the identity evidence"
	s.respond(w, result)
}

func (s *server) resolveKeys(ctx context.Context, agentID string) ([]crypto.PublicKey, error) {
	var response resolveResponse
	if err := s.requestJSON(ctx, http.MethodPost, "/v1alpha1/id/resolve", map[string]string{"id": agentID}, &response); err != nil {
		return nil, err
	}

	if response.ResolverMetadata.ID != agentID {
		return nil, errors.New("resolved identity does not match requested Agent ID")
	}

	keys := make([]crypto.PublicKey, 0, len(response.ResolverMetadata.VerificationMethod))
	for _, method := range response.ResolverMetadata.VerificationMethod {
		publicJWK, err := sanitizePublicJWK(method.PublicKeyJWK)
		if err != nil {
			continue
		}

		key, err := jwk.ParseKey(publicJWK)
		if err != nil {
			continue
		}

		if pub, ok := clientjws.PublicKeyFromJWK(key); ok {
			keys = append(keys, pub)
		}
	}

	if len(keys) == 0 {
		return nil, resolvers.ErrNoKeys
	}

	return keys, nil
}

func (s *server) verifyMatchingAgentBadge(ctx context.Context, agentID, recordCID string) (bool, time.Time, error) {
	path := "/v1alpha1/vc/" + url.PathEscape(agentID) + "/.well-known/vcs.json"

	var credentials wellKnownResponse
	if err := s.requestJSON(ctx, http.MethodGet, path, nil, &credentials); err != nil {
		return false, time.Time{}, err
	}

	for _, credential := range credentials.VCs {
		if credential.Value == "" || credential.EnvelopeType != "CREDENTIAL_ENVELOPE_TYPE_JOSE" {
			continue
		}

		var verified verifyResponse

		body := map[string]any{"vc": map[string]string{
			"envelopeType": credential.EnvelopeType,
			"value":        credential.Value,
		}}
		if err := s.requestJSON(ctx, http.MethodPost, "/v1alpha1/vc/verify", body, &verified); err != nil {
			return false, time.Time{}, err
		}

		if !verified.Status {
			continue
		}

		matches, until, err := badgeMatches(credential.Value, agentID, recordCID)
		if err == nil && matches {
			return true, until, nil
		}
	}

	return false, time.Time{}, nil
}

func (s *server) requestJSON(ctx context.Context, method, path string, input, output any) error {
	var body io.Reader

	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("call Identity Node: %w", err)
		}

		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.identityURL+path, body)
	if err != nil {
		return fmt.Errorf("call Identity Node: %w", err)
	}

	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	response, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("call Identity Node: %w", err)
	}
	defer response.Body.Close() //nolint:errcheck

	if response.StatusCode != http.StatusOK {
		return &safefetch.StatusError{Code: response.StatusCode}
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes+1))
	if err != nil || len(data) > maxBodyBytes {
		return errors.New("invalid Identity Node response")
	}

	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode Identity Node response: %w", err)
	}

	return nil
}

func (s *server) signResult(result agntcy.VerificationResult) (string, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("sign verifier result: %w", err)
	}

	signed, err := jws.Sign(payload, jws.WithKey(jwa.RS256, s.privateKey))
	if err != nil {
		return "", fmt.Errorf("sign verifier result: %w", err)
	}

	return string(signed), nil
}

// parseCanonicalPayload accepts exactly the bytes produced by Claim.GetPayload,
// including the signed role and expiry. The older pipe-delimited form is refused.
func parseCanonicalPayload(payload []byte, signature string) (*identityv1.Claim, error) {
	var input identityv1.CanonicalPayload

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		return nil, fmt.Errorf("parse canonical claim: %w", err)
	}

	role, ok := identityv1.ClaimRole_value[input.Role]
	if !ok || input.RecordCID == "" {
		return nil, errors.New("invalid canonical claim")
	}

	claim := &identityv1.Claim{Role: identityv1.ClaimRole(role), RecordCid: input.RecordCID, Subject: input.Subject, SignedAt: input.SignedAt, Signature: signature}
	if input.ExpiresAt != "" {
		claim.ExpiresAt = &input.ExpiresAt
	}

	canonical, err := claim.GetPayload()
	if err != nil || !bytes.Equal(payload, canonical) {
		return nil, errors.New("non-canonical claim payload")
	}

	if err := clientidentity.Check(claim, input.RecordCID, input.Subject); err != nil {
		return nil, fmt.Errorf("parse canonical claim: %w", err)
	}

	return claim, nil
}

func badgeMatches(compactJOSE, agentID, recordCID string) (bool, time.Time, error) {
	parts := strings.Split(compactJOSE, ".")
	if len(parts) != compactSegments {
		return false, time.Time{}, errors.New("agent badge is not compact JOSE")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false, time.Time{}, fmt.Errorf("parse agent badge: %w", err)
	}

	var credential map[string]any
	if err := json.Unmarshal(payload, &credential); err != nil {
		return false, time.Time{}, fmt.Errorf("parse agent badge: %w", err)
	}

	if !hasCredentialType(credential["type"], "AgentBadge") {
		return false, time.Time{}, nil
	}

	subject, ok := credential["credentialSubject"].(map[string]any)
	if !ok || subject["id"] != agentID {
		return false, time.Time{}, nil
	}

	badge, ok := subject["badge"].(map[string]any)
	if !ok {
		return false, time.Time{}, nil
	}

	badgeStruct, err := structpb.NewStruct(badge)
	if err != nil {
		return false, time.Time{}, fmt.Errorf("parse agent badge: %w", err)
	}

	until, err := credentialValidity(credential, time.Now())
	if err != nil {
		return false, time.Time{}, fmt.Errorf("parse agent badge: %w", err)
	}

	return (&corev1.Record{Data: badgeStruct}).GetCid() == recordCID, until, nil
}

// credentialValidity bounds mutable badge lookup by any credential/JWT expiry.
// Signature validation is performed by the configured Identity Node first.
func credentialValidity(credential map[string]any, now time.Time) (time.Time, error) {
	var until time.Time

	for _, field := range []string{"validUntil", "expirationDate"} {
		if value, ok := credential[field]; ok {
			text, ok := value.(string)
			if !ok {
				return until, errors.New("invalid credential expiry")
			}

			expiry, err := time.Parse(time.RFC3339, text)
			if err != nil || !expiry.After(now) {
				return until, errors.New("expired credential")
			}

			if until.IsZero() || expiry.Before(until) {
				until = expiry
			}
		}
	}

	for _, field := range []string{"validFrom", "issuanceDate"} {
		if value, ok := credential[field]; ok {
			text, ok := value.(string)
			if !ok {
				return until, errors.New("invalid credential start time")
			}

			start, err := time.Parse(time.RFC3339, text)
			if err != nil || start.After(now) {
				return until, errors.New("credential is not yet valid")
			}
		}
	}

	return jwtValidity(credential, now, until)
}

func jwtValidity(credential map[string]any, now, until time.Time) (time.Time, error) {
	if value, ok := credential["exp"]; ok {
		seconds, ok := value.(float64)
		if !ok {
			return until, errors.New("invalid JWT expiry")
		}

		expiry := time.Unix(int64(seconds), 0)
		if !expiry.After(now) {
			return until, errors.New("expired JWT")
		}

		if until.IsZero() || expiry.Before(until) {
			until = expiry
		}
	}

	if value, ok := credential["nbf"]; ok {
		seconds, ok := value.(float64)
		if !ok || time.Unix(int64(seconds), 0).After(now) {
			return until, errors.New("JWT is not yet valid")
		}
	}

	return until, nil
}

func hasCredentialType(value any, expected string) bool {
	switch types := value.(type) {
	case string:
		return types == expected
	case []any:
		for _, item := range types {
			if item == expected {
				return true
			}
		}
	}

	return false
}

func sanitizePublicJWK(input json.RawMessage) ([]byte, error) {
	var fields map[string]any
	if err := json.Unmarshal(input, &fields); err != nil {
		return nil, fmt.Errorf("sanitize agent public key: %w", err)
	}

	for _, name := range []string{"d", "p", "q", "dp", "dq", "qi", "oth", "priv", "seed"} {
		delete(fields, name)
	}

	encoded, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encode public JWK: %w", err)
	}

	return encoded, nil
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, errors.New("RESULT_SIGNING_KEY_PATH is required")
	}

	data, err := os.ReadFile(path) //nolint:gosec // Administrator-provided signing key path; no request data enters the path.
	if err != nil {
		return nil, fmt.Errorf("load signing key: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid private-key PEM")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			if rsaKey.N.BitLen() < minimumRSABits {
				return nil, errors.New("result-signing RSA key must be at least 2048 bits")
			}

			return rsaKey, nil
		}
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("load signing key: %w", err)
	}

	if key.N.BitLen() < minimumRSABits {
		return nil, errors.New("result-signing RSA key must be at least 2048 bits")
	}

	return key, nil
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return fallback
}
