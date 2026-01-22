// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authprovider

import (
	"context"
)

// UserIdentity represents a validated user identity from any authentication provider.
// This is a provider-agnostic representation that can be used by authorization logic
// without knowing the specific provider details.
type UserIdentity struct {
	// Provider is the authentication provider name (github, google, azure, etc.)
	Provider string

	// UserID is the provider-specific unique identifier for the user.
	// Examples: GitHub user ID (numeric), Azure object ID (GUID), Google sub claim
	UserID string

	// Username is the display name or login name.
	// Examples: GitHub login, Azure UPN, Google email
	Username string

	// Email is the user's email address (optional, not all providers guarantee this).
	Email string

	// Attributes contains provider-specific additional information.
	// This allows storing provider-specific data without polluting the common fields.
	// Examples: avatar URL, profile URL, provider-specific roles, etc.
	Attributes map[string]string
}

// OrgConstruct represents a provider-specific organizational unit.
// The term "OrgConstruct" (organizational construct) is deliberately generic to
// accommodate the varying terminology used by different providers:
//
// • GitHub: "organization" - User-created organizations for collaboration
// • Azure: "tenant" - Azure Active Directory tenant
// • Google: "domain" - Google Workspace domain
// • AWS: "account" - AWS account ID
// • Okta: "organization" - Okta org domain
//
// This abstraction allows unified authorization rules across providers.
type OrgConstruct struct {
	// ID is the provider-specific unique identifier.
	// Examples: GitHub org ID (numeric), Azure tenant ID (GUID), Google domain
	ID string

	// Name is the human-readable identifier.
	// Examples:
	// • GitHub: org login ("agntcy")
	// • Azure: tenant domain ("contoso.onmicrosoft.com")
	// • Google: workspace domain ("agntcy.com")
	// • AWS: account ID ("123456789012")
	Name string

	// Type indicates the provider-specific construct type.
	// This helps authorization logic understand what kind of organizational
	// unit this represents.
	//
	// Standard values:
	// • "github-org" - GitHub organization
	// • "azure-tenant" - Azure AD tenant
	// • "google-domain" - Google Workspace domain
	// • "aws-account" - AWS account
	// • "okta-org" - Okta organization
	Type string
}

// Provider defines the interface for external authentication providers.
// This abstraction allows supporting multiple identity providers (GitHub, Google, Azure, etc.)
// with a unified interface for credential validation and organizational construct extraction.
type Provider interface {
	// Name returns the provider identifier (e.g., "github", "google", "azure").
	Name() string

	// ValidateToken validates the provider-specific credential and returns user identity.
	// The token format varies by provider (OAuth token, JWT, etc.).
	ValidateToken(ctx context.Context, token string) (*UserIdentity, error)

	// GetOrgConstructs returns the user's organizational affiliations.
	// The meaning of "org construct" varies by provider:
	// • GitHub: organizations
	// • Azure: tenants
	// • Google: domains
	// • AWS: accounts
	GetOrgConstructs(ctx context.Context, token string) ([]OrgConstruct, error)
}

// ProviderRegistry holds registered authentication providers.
// This allows runtime provider selection based on configuration.
type ProviderRegistry struct {
	providers map[string]Provider
}

// NewProviderRegistry creates a new provider registry.
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *ProviderRegistry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name.
func (r *ProviderRegistry) Get(name string) (Provider, bool) {
	provider, ok := r.providers[name]

	return provider, ok
}

// List returns all registered provider names.
func (r *ProviderRegistry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}

	return names
}
