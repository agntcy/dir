// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/agntcy/dir/pkg/authprovider"
	"github.com/google/go-github/v50/github"
	"golang.org/x/oauth2"
)

const (
	// ProviderNameGithub is the identifier for the GitHub provider.
	ProviderNameGithub = "github"

	// Default configuration values.
	defaultCacheTTL   = 5 * time.Minute
	defaultAPITimeout = 10 * time.Second

	// GitHub API pagination settings.
	githubPerPage = 100
)

// Provider implements authprovider.Provider for GitHub.
// It validates GitHub Personal Access Tokens (PATs) and OAuth tokens,
// fetches user information, and retrieves organization memberships.
type Provider struct {
	// cache stores validated user identities and org constructs
	cache      map[string]*cacheEntry
	cacheMu    sync.RWMutex
	cacheTTL   time.Duration
	apiTimeout time.Duration
}

// cacheEntry holds cached provider data.
type cacheEntry struct {
	identity      *authprovider.UserIdentity
	orgConstructs []authprovider.OrgConstruct
	expiresAt     time.Time
}

// Config holds GitHub provider configuration.
type Config struct {
	// CacheTTL is how long to cache GitHub API responses
	// Default: 5 minutes
	CacheTTL time.Duration

	// APITimeout is the timeout for GitHub API calls
	// Default: 10 seconds
	APITimeout time.Duration
}

// DefaultConfig returns default configuration.
func DefaultConfig() *Config {
	return &Config{
		CacheTTL:   defaultCacheTTL,
		APITimeout: defaultAPITimeout,
	}
}

// NewProvider creates a new GitHub authentication provider.
func NewProvider(cfg *Config) *Provider {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Provider{
		cache:      make(map[string]*cacheEntry),
		cacheTTL:   cfg.CacheTTL,
		apiTimeout: cfg.APITimeout,
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return ProviderNameGithub
}

// ValidateToken validates a GitHub PAT or OAuth token and returns user identity.
// This method checks the cache first to minimize GitHub API calls.
func (p *Provider) ValidateToken(ctx context.Context, token string) (*authprovider.UserIdentity, error) {
	// Check cache first
	p.cacheMu.RLock()

	if entry, ok := p.cache[token]; ok && time.Now().Before(entry.expiresAt) {
		p.cacheMu.RUnlock()

		return entry.identity, nil
	}

	p.cacheMu.RUnlock()

	// Create GitHub client with official SDK
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Set timeout
	clientCtx, cancel := context.WithTimeout(ctx, p.apiTimeout)
	defer cancel()

	// Call GitHub API to get authenticated user
	user, resp, err := client.Users.Get(clientCtx, "")
	if err != nil {
		// Provide helpful error messages
		if resp != nil {
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				return nil, errors.New("invalid or expired GitHub token")
			case http.StatusForbidden:
				return nil, errors.New("GitHub token lacks required permissions (need: read:user, read:org)")
			case http.StatusTooManyRequests:
				return nil, errors.New("GitHub API rate limit exceeded")
			}
		}

		return nil, fmt.Errorf("GitHub API error: %w", err)
	}

	// Convert to generic identity
	identity := &authprovider.UserIdentity{
		Provider: "github",
		UserID:   strconv.FormatInt(user.GetID(), 10),
		Username: user.GetLogin(),
		Email:    user.GetEmail(),
		Attributes: map[string]string{
			"github_id":   strconv.FormatInt(user.GetID(), 10),
			"avatar_url":  user.GetAvatarURL(),
			"profile_url": user.GetHTMLURL(),
			"name":        user.GetName(),
		},
	}

	// Cache the identity
	p.cacheMu.Lock()

	if p.cache[token] == nil {
		p.cache[token] = &cacheEntry{}
	}

	p.cache[token].identity = identity
	p.cache[token].expiresAt = time.Now().Add(p.cacheTTL)
	p.cacheMu.Unlock()

	return identity, nil
}

// GetOrgConstructs fetches the user's GitHub organizations.
// This method also uses caching to minimize GitHub API calls.
func (p *Provider) GetOrgConstructs(ctx context.Context, token string) ([]authprovider.OrgConstruct, error) {
	// Check cache first
	p.cacheMu.RLock()

	if entry, ok := p.cache[token]; ok && time.Now().Before(entry.expiresAt) && entry.orgConstructs != nil {
		p.cacheMu.RUnlock()

		return entry.orgConstructs, nil
	}

	p.cacheMu.RUnlock()

	// Create GitHub client
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Set timeout
	clientCtx, cancel := context.WithTimeout(ctx, p.apiTimeout)
	defer cancel()

	// Fetch all organizations (with pagination)
	opts := &github.ListOptions{PerPage: githubPerPage}

	var allOrgs []*github.Organization

	for {
		orgs, resp, err := client.Organizations.List(clientCtx, "", opts)
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusForbidden {
				return nil, errors.New("GitHub token lacks read:org permission")
			}

			return nil, fmt.Errorf("failed to fetch organizations: %w", err)
		}

		allOrgs = append(allOrgs, orgs...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	// Convert to generic OrgConstruct structure
	result := make([]authprovider.OrgConstruct, len(allOrgs))
	for i, org := range allOrgs {
		result[i] = authprovider.OrgConstruct{
			ID:   strconv.FormatInt(org.GetID(), 10),
			Name: org.GetLogin(), // e.g., "agntcy"
			Type: "github-org",
		}
	}

	// Cache the org constructs
	p.cacheMu.Lock()

	if p.cache[token] == nil {
		p.cache[token] = &cacheEntry{}
	}

	p.cache[token].orgConstructs = result
	p.cache[token].expiresAt = time.Now().Add(p.cacheTTL)
	p.cacheMu.Unlock()

	return result, nil
}

// IsMemberOfOrgConstruct checks if the user is a member of a specific org construct.
// This is a convenience method for authorization checks.
func (p *Provider) IsMemberOfOrgConstruct(ctx context.Context, token string, orgName string) (bool, error) {
	orgConstructs, err := p.GetOrgConstructs(ctx, token)
	if err != nil {
		return false, err
	}

	for _, oc := range orgConstructs {
		if oc.Name == orgName {
			return true, nil
		}
	}

	return false, nil
}

// IsMemberOfAnyOrgConstruct checks if the user is a member of any of the specified org constructs.
// Returns true and the matched org name if found.
func (p *Provider) IsMemberOfAnyOrgConstruct(ctx context.Context, token string, allowedOrgs []string) (bool, string, error) {
	orgConstructs, err := p.GetOrgConstructs(ctx, token)
	if err != nil {
		return false, "", err
	}

	// Create lookup map
	orgMap := make(map[string]bool)
	for _, oc := range orgConstructs {
		orgMap[oc.Name] = true
	}

	// Check against allowed list
	for _, allowed := range allowedOrgs {
		if orgMap[allowed] {
			return true, allowed, nil
		}
	}

	return false, "", nil
}

// ClearCache removes all cached entries.
// Useful for testing or when you want to force fresh GitHub API calls.
func (p *Provider) ClearCache() {
	p.cacheMu.Lock()
	p.cache = make(map[string]*cacheEntry)
	p.cacheMu.Unlock()
}

// ClearTokenCache removes the cache entry for a specific token.
func (p *Provider) ClearTokenCache(token string) {
	p.cacheMu.Lock()
	delete(p.cache, token)
	p.cacheMu.Unlock()
}
