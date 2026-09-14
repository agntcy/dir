// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ans

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify"
	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	ansconfig "github.com/agntcy/dir/server/identity/ans/config"
	"github.com/agntcy/dir/utils/logging"
)

const (
	// statusTokenClockSkew is the tolerance applied to the status token expiry.
	statusTokenClockSkew = 30 * time.Second

	// fetchBudgetDivisor bounds one transparency-log fetch to the verification
	// budget divided by this, so a stalled request cannot consume the time the
	// remaining fetches need.
	fetchBudgetDivisor = 2

	// rootKeysTTL is how long fetched root keys serve a log origin before they
	// are fetched again, in unpinned mode.
	rootKeysTTL = 10 * time.Minute
)

var logger = logging.Logger("identity/ans")

// Resolver verifies ans:// claim signatures through the Agent Name Service.
// The claim carries the signer's identity certificate; the certificate is
// trusted only once the agent's transparency log, found through DNS and
// checked against the allow-list, attests its fingerprint for the agent the
// subject names. It implements identity.Resolver and is safe for concurrent
// use.
type Resolver struct {
	cfg          ansconfig.Config
	trustedHosts map[string]struct{}
	pinnedKeys   *scitt.KeyStore
	transport    *http.Transport
	badges       badgeFinder
	newLogClient func(base string) (scitt.Client, error)
	clock        scitt.ClockFunc
	breaker      *breaker

	// mu guards the per-origin caches and is never held across a fetch.
	mu       sync.Mutex
	clients  map[string]scitt.Client
	rootKeys map[string]cachedRootKeys
}

// cachedRootKeys is one log origin's fetched key set and when it stops
// serving.
type cachedRootKeys struct {
	keys    *scitt.KeyStore
	expires time.Time
}

// New builds a resolver from an enabled configuration. The configuration is
// validated here, pinned root keys are parsed so a malformed line fails
// startup, and one HTTP transport with ca_file appended to the system roots
// is shared by every log connection. No network call is made.
func New(cfg ansconfig.Config) (*Resolver, error) {
	if !cfg.Enabled {
		return nil, errors.New("ans: the resolver needs an enabled configuration")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err //nolint:wrapcheck // the configuration errors already name this component
	}

	trusted := make(map[string]struct{}, len(cfg.TrustedLogHosts))
	for _, host := range cfg.TrustedLogHosts {
		trusted[host] = struct{}{}
	}

	r := &Resolver{
		cfg:          cfg,
		trustedHosts: trusted,
		clock:        time.Now,
		breaker:      newBreaker(),
		clients:      make(map[string]scitt.Client),
		rootKeys:     make(map[string]cachedRootKeys),
	}

	pinned := 0

	if len(cfg.RootKeys) > 0 {
		keys, err := scitt.NewKeyStore(cfg.RootKeys)
		if err != nil {
			return nil, fmt.Errorf("ans: root_keys: %w", err)
		}

		r.pinnedKeys = keys
		pinned = keys.Len()
	}

	transport, err := newTransport(cfg.CAFile)
	if err != nil {
		return nil, err
	}

	r.transport = transport
	r.badges = defaultResolver(cfg.DNSServer)
	r.newLogClient = r.httpLogClient

	logger.Info("ANS identity resolver configured",
		"trustedLogHosts", cfg.TrustedLogHosts,
		"pinnedRootKeys", pinned,
		"unpinned", r.pinnedKeys == nil,
		"timeout", cfg.GetTimeout(),
		"dnsServer", cfg.DNSServer,
		"caFile", cfg.CAFile != "")

	return r, nil
}

// Scheme implements identity.Resolver.
func (*Resolver) Scheme() string {
	return "ans"
}

// Verify implements identity.Resolver. It runs the trust path for one claim
// within the configured time budget. Without a network call: the subject
// must be a canonical ANS name; the JWS must carry the signer's identity
// certificate as x5c; the certificate must name the agent, be valid now, and
// hold a key claims are verified with; and the signature must verify against
// that key, otherwise the result is the verdict (false, nil). Only then the
// certificate is anchored: the agent's badge record in DNS names an
// allow-listed transparency log whose status token attests the certificate's
// fingerprint for the agent and whose receipt proves it logged the agent.
// Every failure other than the signature verdict is an error whose text
// starts with the stage that failed and stays stable, or the caller's own
// context error when that context is done.
func (r *Resolver) Verify(ctx context.Context, subject string, signature, payload []byte) (bool, error) {
	budget, cancel := context.WithTimeout(ctx, r.cfg.GetTimeout())
	defer cancel()

	want, err := parseAgentName(subject)
	if err != nil {
		logger.Debug("Claim subject rejected", "subject", truncate(subject), "error", err)

		return false, err
	}

	cert, err := identityv1.CertificateFromJWS(string(signature))
	if err != nil {
		logger.Debug("Claim carries no usable certificate", "ansName", want.String(), "error", err)

		return false, failWith(stageCertificate, err, err.Error())
	}

	fingerprint := verify.CertFingerprintFromDER(cert.Raw)

	if err := checkCertificate(cert, want, r.clock()); err != nil {
		logger.Debug("Claim certificate rejected", "ansName", want.String(), "fingerprint", fingerprint.String(), "error", err)

		return false, err
	}

	if err := identityv1.VerifyJWS(string(signature), cert.PublicKey, payload); err != nil {
		logger.Debug("Claim signature does not verify against the certificate it carries", "ansName", want.String(), "fingerprint", fingerprint.String(), "error", err)

		return false, nil //nolint:nilerr // per identity.Resolver, a signature that does not verify is a verdict, not an error
	}

	if err := r.verifyAttestation(budget, want, cert); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return false, ctxErr //nolint:wrapcheck // the caller's own context error, returned as is so a task can recognize it
		}

		logger.Debug("Claim certificate is not attested", "ansName", want.String(), "fingerprint", fingerprint.String(), "error", err)

		return false, err
	}

	logger.Debug("Claim verified", "ansName", want.String(), "fingerprint", fingerprint.String())

	return true, nil
}

// verifyAttestation anchors cert for want through the Agent Name Service:
// the agent's badge record in DNS names its transparency log, which must be
// allow-listed and not under an open circuit; the log's status token,
// verified against the trusted keys, must name the agent, allow connections,
// and list cert's fingerprint among the agent's identity certificates; and
// the log's receipt must prove it signed an event for the same agent.
func (r *Resolver) verifyAttestation(ctx context.Context, want agentName, cert *x509.Certificate) error {
	target, err := r.resolveTarget(ctx, want)
	if err != nil {
		return err
	}

	if until, open := r.breaker.openUntil(target.LogHost, r.clock()); open {
		return fail(stageLog, fmt.Sprintf("transparency log %s is unavailable until %s; circuit open", target.LogHost, until.UTC().Format(time.RFC3339)))
	}

	client, err := r.logClient(target)
	if err != nil {
		return err
	}

	if err := r.verifyStatusToken(ctx, client, target, want, verify.CertFingerprintFromDER(cert.Raw)); err != nil {
		return err
	}

	return r.verifyReceipt(ctx, client, target, want)
}

// resolveTarget finds the agent's badge record in DNS and checks the URL it
// carries against the allow-list before any request is made.
func (r *Resolver) resolveTarget(ctx context.Context, want agentName) (badgeTarget, error) {
	record, err := r.badges.FindBadgeForVersion(ctx, want.fqdn, want.version)
	if err != nil {
		if errors.Is(err, verify.ErrRecordNotFound) {
			logger.Debug("No badge record for this agent", "ansName", want.String())

			return badgeTarget{}, fail(stageDNS, fmt.Sprintf("no _ans-badge record for %s version %s", want.host, want.version))
		}

		logger.Warn("Badge lookup failed", "ansName", want.String(), "error", err)

		return badgeTarget{}, failWith(stageDNS, err, describe(err))
	}

	logger.Debug("Resolved badge record", "ansName", want.String(), "source", badgeSourceName(record.Source), "url", truncate(record.URL))

	target, err := parseBadgeURL(record.URL)
	if err != nil {
		logger.Debug("Badge URL rejected", "ansName", want.String(), "error", err)

		return badgeTarget{}, err
	}

	if _, ok := r.trustedHosts[target.LogHost]; !ok {
		logger.Debug("Badge points at an untrusted transparency log", "ansName", want.String(), "logHost", truncate(target.LogHost))

		return badgeTarget{}, fail(stageBadgeURL, fmt.Sprintf("log host %q is not a trusted transparency log", truncate(target.LogHost)))
	}

	return target, nil
}

// keysFor returns the key set that authenticates the target's log: the
// pinned root keys, or in unpinned mode the keys fetched from the log, cached
// per origin for rootKeysTTL.
func (r *Resolver) keysFor(ctx context.Context, client scitt.Client, target badgeTarget) (scitt.KeyLookup, error) {
	if r.pinnedKeys != nil {
		return r.pinnedKeys, nil
	}

	if keys, ok := r.cachedRootKeys(target.LogBase); ok {
		return keys, nil
	}

	lines, err := client.FetchRootKeys(ctx)
	if err != nil {
		return nil, failWith(stageRootKeys, err, describe(err))
	}

	if len(lines) == 0 {
		logger.Warn("Transparency log served no root keys", "logBase", target.LogBase)

		return nil, fail(stageRootKeys, "transparency log served no root keys")
	}

	keys, err := scitt.NewKeyStore(lines)
	if err != nil {
		logger.Warn("Transparency log served malformed root keys", "logBase", target.LogBase, "error", err)

		return nil, failWith(stageRootKeys, err, "transparency log served malformed root keys")
	}

	r.storeRootKeys(target.LogBase, keys)

	logger.Debug("Fetched root keys", "logBase", target.LogBase, "keys", keys.Len())

	return keys, nil
}

func (r *Resolver) cachedRootKeys(origin string) (*scitt.KeyStore, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.rootKeys[origin]
	if !ok || !r.clock().Before(entry.expires) {
		return nil, false
	}

	return entry.keys, true
}

func (r *Resolver) storeRootKeys(origin string, keys *scitt.KeyStore) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rootKeys[origin] = cachedRootKeys{keys: keys, expires: r.clock().Add(rootKeysTTL)}
}

func (r *Resolver) dropRootKeys(origin string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.rootKeys, origin)
}

// verifyWithLogKeys runs check with the keys that authenticate the target's
// log. A signature by a key the set does not hold is, in unpinned mode, the
// sign of a log that rotated its keys since they were fetched: the cached set
// is dropped and check runs once more against fresh keys, so the rotation
// heals within one verification. In pinned mode it is reported at WARN, since
// only the operator can pin the new key.
func (r *Resolver) verifyWithLogKeys(ctx context.Context, client scitt.Client, target badgeTarget, check func(scitt.KeyLookup) error) error {
	keys, err := r.keysFor(ctx, client, target)
	if err != nil {
		return err
	}

	err = check(keys)

	kid, unknown := unknownKeyID(err)
	if !unknown {
		return err
	}

	if r.pinnedKeys != nil {
		logger.Warn("Transparency log signed with a key that is not pinned", "kid", hex.EncodeToString(kid[:]), "logHost", target.LogHost)

		return err
	}

	logger.Debug("Transparency log signed with a key the cached root keys do not hold; fetching them again", "kid", hex.EncodeToString(kid[:]), "logBase", target.LogBase)

	r.dropRootKeys(target.LogBase)

	keys, err = r.keysFor(ctx, client, target)
	if err != nil {
		return err
	}

	return check(keys)
}

// verifyStatusToken fetches and verifies the agent's status token and checks
// that it names this agent, allows connections, and attests the certificate.
func (r *Resolver) verifyStatusToken(ctx context.Context, client scitt.Client, target badgeTarget, want agentName, fingerprint verify.CertFingerprint) error {
	tokenBytes, err := client.FetchStatusToken(ctx, target.AgentID)
	if err != nil {
		return failWith(stageStatusToken, err, describe(err))
	}

	var token *scitt.VerifiedStatusToken

	err = r.verifyWithLogKeys(ctx, client, target, func(keys scitt.KeyLookup) error {
		verified, verifyErr := scitt.VerifyStatusTokenAt(tokenBytes, keys, statusTokenClockSkew, r.clock().Unix())
		if verifyErr != nil {
			return failWith(stageStatusToken, verifyErr, describe(verifyErr))
		}

		token = verified

		return nil
	})
	if err != nil {
		logger.Debug("Status token not verified", "agentId", target.AgentID, "logBase", target.LogBase, "error", err)

		return err
	}

	payload := &token.Payload

	if !strings.EqualFold(payload.AgentID, target.AgentID) {
		return fail(stageStatusToken, fmt.Sprintf("token names agent %s, expected %s", truncate(payload.AgentID), target.AgentID))
	}

	if !want.matches(payload.AnsName) {
		return fail(stageStatusToken, fmt.Sprintf("token names %s, expected %s", truncate(payload.AnsName), want))
	}

	if !payload.Status.IsValidForConnection() {
		return fail(stageStatusToken, fmt.Sprintf("agent status %s does not allow connections", truncate(string(payload.Status))))
	}

	if !scitt.MatchesIdentityCert(payload, fingerprint.Bytes()) {
		logger.Debug("Certificate is not attested by the status token",
			"agentId", target.AgentID,
			"logBase", target.LogBase,
			"fingerprint", fingerprint.String(),
			"identityCerts", len(payload.ValidIdentityCerts))

		return fail(stageCertificate, "certificate is not attested for this agent")
	}

	logger.Debug("Status token verified",
		"agentId", target.AgentID,
		"logBase", target.LogBase,
		"status", payload.Status,
		"fingerprint", fingerprint.String())

	return nil
}

// verifyReceipt fetches and verifies the agent's receipt and checks that the
// logged event names this agent.
//
// The receipt signature covers the event and the protected header. Tree
// size, leaf index and the inclusion path travel in the unsigned COSE header;
// the SDK checks that they are well-formed and walks the path to a root it
// compares with nothing. The receipt therefore proves that the log signed
// this event and nothing about the event's position in the tree.
func (r *Resolver) verifyReceipt(ctx context.Context, client scitt.Client, target badgeTarget, want agentName) error {
	receiptBytes, err := client.FetchReceipt(ctx, target.AgentID)
	if err != nil {
		return failWith(stageReceipt, err, describe(err))
	}

	var receipt *scitt.VerifiedReceipt

	err = r.verifyWithLogKeys(ctx, client, target, func(keys scitt.KeyLookup) error {
		verified, verifyErr := scitt.VerifyReceipt(receiptBytes, keys)
		if verifyErr != nil {
			return failWith(stageReceipt, verifyErr, describe(verifyErr))
		}

		receipt = verified

		return nil
	})
	if err != nil {
		logger.Debug("Receipt not verified", "agentId", target.AgentID, "logBase", target.LogBase, "error", err)

		return err
	}

	event, err := decodeEvent(receipt.EventBytes)
	if err != nil {
		return failWith(stageReceipt, err, "event payload is not an ANS event envelope")
	}

	if !strings.EqualFold(event.agentID(), target.AgentID) {
		return fail(stageReceipt, fmt.Sprintf("receipt event names agent %s, expected %s", truncate(event.agentID()), target.AgentID))
	}

	if !want.matches(event.AnsName) {
		return fail(stageReceipt, fmt.Sprintf("receipt event names %s, expected %s", truncate(event.AnsName), want))
	}

	logger.Debug("Receipt verified",
		"agentId", target.AgentID,
		"logBase", target.LogBase,
		"treeSize", receipt.TreeSize,
		"leafIndex", receipt.LeafIndex)

	return nil
}
