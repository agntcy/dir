package did

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ed25519"
)

// DIDManager interface defines methods for DID registration and validation
type DIDManager interface {
	Register(ctx context.Context, resource string) (*DIDDocument, error)
	Validate(ctx context.Context, didStr string) (bool, error)
	GetResource(ctx context.Context, didStr string) (string, error)
	ResolveDID(ctx context.Context, didStr string) (*DIDDocument, error)
}

// DIDDocument represents a DID document structure
type DIDDocument struct {
	Context              []string             `json:"@context"`
	ID                   string               `json:"id"`
	AlsoKnownAs          []string             `json:"alsoKnownAs,omitempty"`
	VerificationMethod   []VerificationMethod `json:"verificationMethod"`
	Authentication       []string             `json:"authentication"`
	AssertionMethod      []string             `json:"assertionMethod"`
	KeyAgreement         []string             `json:"keyAgreement,omitempty"`
	CapabilityInvocation []string             `json:"capabilityInvocation"`
	CapabilityDelegation []string             `json:"capabilityDelegation,omitempty"`
	Service              []Service            `json:"service,omitempty"`
}

// VerificationMethod represents a verification method in a DID document
type VerificationMethod struct {
	ID                 string `json:"id"`
	Type               string `json:"type"`
	Controller         string `json:"controller"`
	PublicKeyMultibase string `json:"publicKeyMultibase"`
}

// Service represents a service endpoint in a DID document
type Service struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	ServiceEndpoint string `json:"serviceEndpoint"`
}

// Manager implements DIDManager interface
type Manager struct {
	pdsURL   string
	client   *http.Client
	registry map[string]string // DID -> resource mapping (in-memory for simplicity)
}

// NewManager creates a new DID manager with PDS integration
func NewManager(pdsURL string) *Manager {
	// Ensure PDS URL has proper scheme
	if !strings.HasPrefix(pdsURL, "http://") && !strings.HasPrefix(pdsURL, "https://") {
		pdsURL = "https://" + pdsURL
	}

	return &Manager{
		pdsURL:   pdsURL,
		client:   &http.Client{Timeout: 30 * time.Second},
		registry: make(map[string]string),
	}
}

// Register creates and registers a new DID for the given resource with the PDS
func (m *Manager) Register(ctx context.Context, resource string) (*DIDDocument, error) {
	if resource == "" {
		return nil, fmt.Errorf("resource cannot be empty")
	}

	// Generate cryptographic key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate DID using PLC method (AT Protocol standard)
	did := m.generatePLCDID(resource, pubKey)

	// Create DID document
	didDoc := &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/multikey/v1",
		},
		ID: did,
		VerificationMethod: []VerificationMethod{
			{
				ID:                 did + "#atproto",
				Type:               "Multikey",
				Controller:         did,
				PublicKeyMultibase: m.encodePublicKey(pubKey),
			},
		},
		Authentication:       []string{did + "#atproto"},
		AssertionMethod:      []string{did + "#atproto"},
		CapabilityInvocation: []string{did + "#atproto"},
		Service: []Service{
			{
				ID:              did + "#atproto_pds",
				Type:            "AtprotoPersonalDataServer",
				ServiceEndpoint: m.pdsURL,
			},
		},
	}

	// Register with PDS
	err = m.registerWithPDS(ctx, didDoc, privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to register with PDS: %w", err)
	}

	// Store in local registry
	m.registry[did] = resource

	return didDoc, nil
}

// Validate checks if a DID is valid and exists in the PDS
func (m *Manager) Validate(ctx context.Context, didStr string) (bool, error) {
	if didStr == "" {
		return false, fmt.Errorf("DID cannot be empty")
	}

	// Basic DID format validation
	if !strings.HasPrefix(didStr, "did:") {
		return false, nil
	}

	// Try to resolve DID through PDS
	_, err := m.ResolveDID(ctx, didStr)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// GetResource retrieves the resource associated with a DID
func (m *Manager) GetResource(ctx context.Context, didStr string) (string, error) {
	resource, exists := m.registry[didStr]
	if !exists {
		return "", fmt.Errorf("DID not found in local registry: %s", didStr)
	}

	return resource, nil
}

// ResolveDID resolves a DID document from the PDS
func (m *Manager) ResolveDID(ctx context.Context, didStr string) (*DIDDocument, error) {
	// Construct PDS resolution URL
	resolveURL := fmt.Sprintf("%s/xrpc/com.atproto.identity.resolveHandle", m.pdsURL)

	// For DID resolution, we need to extract the handle or use DID resolution endpoint
	// This is a simplified implementation - in production, use proper AT Protocol resolution
	req, err := http.NewRequestWithContext(ctx, "GET", resolveURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("handle", didStr)
	req.URL.RawQuery = q.Encode()

	// Add AT Protocol headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "dir-server/1.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve DID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("DID not found: %s", didStr)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PDS returned error: %d", resp.StatusCode)
	}

	// Parse response - this would be more complex in a real implementation
	// For now, return a basic DID document structure
	didDoc := &DIDDocument{
		Context: []string{
			"https://www.w3.org/ns/did/v1",
			"https://w3id.org/security/multikey/v1",
		},
		ID: didStr,
		Service: []Service{
			{
				ID:              didStr + "#atproto_pds",
				Type:            "AtprotoPersonalDataServer",
				ServiceEndpoint: m.pdsURL,
			},
		},
	}

	return didDoc, nil
}

// generatePLCDID generates a DID using the PLC method
func (m *Manager) generatePLCDID(resource string, pubKey ed25519.PublicKey) string {
	// Create genesis data for PLC DID
	genesis := map[string]interface{}{
		"type":      "plc_operation",
		"operation": "create",
		"resource":  resource,
		"publicKey": hex.EncodeToString(pubKey),
		"timestamp": time.Now().Unix(),
		"nonce":     uuid.New().String(),
	}

	// Serialize and hash
	genesisBytes, _ := json.Marshal(genesis)
	hash := sha256.Sum256(genesisBytes)

	// Create PLC identifier (first 24 bytes of hash, base32 encoded)
	plcID := strings.ToLower(hex.EncodeToString(hash[:12]))

	return fmt.Sprintf("did:plc:%s", plcID)
}

// encodePublicKey encodes a public key in multibase format
func (m *Manager) encodePublicKey(pubKey ed25519.PublicKey) string {
	// Ed25519 multicodec prefix is 0xed01
	multicodec := append([]byte{0xed, 0x01}, pubKey...)
	// Use base58btc encoding (prefix 'z')
	return "z" + m.base58Encode(multicodec)
}

// registerWithPDS registers the DID document with the PDS
func (m *Manager) registerWithPDS(ctx context.Context, didDoc *DIDDocument, privKey ed25519.PrivateKey) error {
	// Construct PDS registration URL
	registerURL := fmt.Sprintf("%s/xrpc/com.atproto.server.createAccount", m.pdsURL)

	// Create registration payload
	payload := map[string]interface{}{
		"handle":   didDoc.ID,
		"email":    fmt.Sprintf("did@%s", strings.ReplaceAll(didDoc.ID, ":", "-")),
		"password": "temp-password", // In production, use proper authentication
		"didDoc":   didDoc,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registration payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", registerURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "dir-server/1.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register with PDS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("PDS registration failed with status: %d", resp.StatusCode)
	}

	return nil
}

// base58Encode provides a simple base58 encoding (simplified implementation)
func (m *Manager) base58Encode(data []byte) string {
	alphabet := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	// Count leading zeros
	zeros := 0
	for i := 0; i < len(data) && data[i] == 0; i++ {
		zeros++
	}

	// Convert to big integer and encode
	result := make([]byte, 0, len(data)*2)

	// Simple base conversion (this is a simplified version)
	for _, b := range data {
		if b == 0 && len(result) == 0 {
			continue
		}
		result = append(result, alphabet[int(b)%len(alphabet)])
	}

	// Add leading '1's for leading zeros
	for i := 0; i < zeros; i++ {
		result = append([]byte{'1'}, result...)
	}

	return string(result)
}

// Legacy functions for backward compatibility
func register() {
	// Deprecated: Use Manager.Register() instead
}

func fetch() {
	// Deprecated: Use Manager.ResolveDID() instead
}
