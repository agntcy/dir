// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package authzserver implements Envoy's External Authorization gRPC API
// for validating external authentication tokens (GitHub, Google, Azure, etc.)
// and enforcing authorization rules.
package authzserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/agntcy/dir/auth/authprovider"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
)

const (
	// authHeaderParts is the expected number of parts in "Bearer <token>" header.
	authHeaderParts = 2
)

// AuthorizationServer implements the Envoy ext_authz gRPC API.
// It supports multiple authentication providers through a generic interface.
type AuthorizationServer struct {
	authv3.UnimplementedAuthorizationServer

	providers       map[string]authprovider.Provider
	roleResolver    *RoleResolver
	logger          *slog.Logger
	defaultProvider string
}

// Config holds the authorization server configuration.
type Config struct {
	// DefaultProvider is used when provider cannot be auto-detected
	DefaultProvider string

	// Roles defines the available roles and their permissions.
	// Key is the role name, value is the Role definition.
	Roles map[string]Role

	// DefaultRole is assigned to authenticated users who don't have an explicit role.
	// Empty string means deny by default.
	DefaultRole string

	// UserDenyList explicitly denies specific users (takes precedence over all roles).
	// Format: "provider:username" (e.g., "github:blocked-user")
	UserDenyList []string
}

// Role defines a set of permissions and the users/orgs assigned to it.
type Role struct {
	// AllowedMethods is a list of gRPC methods this role can access.
	// Use "*" for wildcard (all methods).
	// Format: "/package.Service/Method" (e.g., "/store.StoreService/Push")
	AllowedMethods []string

	// Orgs is a list of organization names assigned to this role.
	// Users in these orgs will have this role's permissions.
	Orgs []string

	// Users is a list of specific users assigned to this role.
	// Format: "provider:username" (e.g., "github:alice")
	// User assignments take precedence over org assignments.
	Users []string
}

// NewAuthorizationServer creates a new authorization server with multiple providers.
func NewAuthorizationServer(
	providers map[string]authprovider.Provider,
	config *Config,
	logger *slog.Logger,
) (*AuthorizationServer, error) {
	if config == nil {
		config = &Config{}
	}

	if logger == nil {
		logger = slog.Default()
	}

	if config.DefaultProvider == "" {
		config.DefaultProvider = authprovider.ProviderGithub
	}

	// Create role resolver for RBAC (Casbin-based)
	roleResolver, err := NewRoleResolver(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create role resolver: %w", err)
	}

	return &AuthorizationServer{
		providers:       providers,
		roleResolver:    roleResolver,
		logger:          logger,
		defaultProvider: config.DefaultProvider,
	}, nil
}

// Check implements the ext_authz Check RPC.
func (s *AuthorizationServer) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	httpReq := req.GetAttributes().GetRequest().GetHttp()

	s.logger.Debug("received authorization request",
		"path", httpReq.GetPath(),
		"method", httpReq.GetMethod(),
	)

	// Extract Authorization header
	authHeader := httpReq.GetHeaders()["authorization"]
	if authHeader == "" {
		return s.denyResponse(codes.Unauthenticated, "missing Authorization header"), nil
	}

	// Parse Bearer token
	token, err := extractBearerToken(authHeader)
	if err != nil {
		return s.denyResponse(codes.Unauthenticated, err.Error()), nil
	}

	// Detect provider (combination approach)
	providerName := s.detectProvider(httpReq, token)

	s.logger.Debug("detected provider", "provider", providerName)

	// Get provider
	provider, ok := s.providers[providerName]
	if !ok {
		return s.denyResponse(
			codes.Unavailable,
			fmt.Sprintf("provider %s is not configured", providerName),
		), nil
	}

	// Validate token using provider
	identity, err := provider.ValidateToken(ctx, token)
	if err != nil {
		s.logger.Warn("token validation failed",
			"provider", providerName,
			"error", err,
		)

		return s.denyResponse(codes.Unauthenticated, "invalid token: "+err.Error()), nil
	}

	// Get org constructs
	orgConstructs, err := provider.GetOrgConstructs(ctx, token)
	if err != nil {
		s.logger.Warn("failed to fetch org constructs",
			"provider", providerName,
			"user", identity.Username,
			"error", err,
		)
		// Continue without org info (might be allowed anyway)
		orgConstructs = []authprovider.OrgConstruct{}
	}

	// Extract API method from request path
	apiMethod := httpReq.GetPath() // e.g., "/store.StoreService/Push"

	// Check authorization rules (role-based) - delegate to RoleResolver
	userKey := fmt.Sprintf("%s:%s", identity.Provider, identity.Username)

	userOrgs := extractOrgNames(orgConstructs)
	if err := s.roleResolver.Authorize(identity.Username, userKey, userOrgs, apiMethod); err != nil {
		s.logger.Info("authorization denied",
			"provider", identity.Provider,
			"user", identity.Username,
			"org_constructs", userOrgs,
			"method", apiMethod,
			"reason", err.Error(),
		)

		return s.denyResponse(codes.PermissionDenied, err.Error()), nil
	}

	s.logger.Info("authorization granted",
		"provider", identity.Provider,
		"user", identity.Username,
		"org_constructs", userOrgs,
		"method", apiMethod,
	)

	return s.allowResponse(identity, orgConstructs), nil
}

// detectProvider determines which provider to use based on request context.
// Priority: 1. Header, 2. Token format, 3. Default.
func (s *AuthorizationServer) detectProvider(httpReq *authv3.AttributeContext_HttpRequest, token string) string {
	// Priority 1: Explicit header (user override)
	if provider := httpReq.GetHeaders()["x-auth-provider"]; provider != "" {
		return provider
	}

	// Priority 2: Token format detection (automatic)
	// GitHub OAuth2 tokens
	if strings.HasPrefix(token, "gho_") || // OAuth token
		strings.HasPrefix(token, "ghu_") || // User token
		strings.HasPrefix(token, "ghs_") || // Server token
		strings.HasPrefix(token, "ghr_") { // Refresh token
		return authprovider.ProviderGithub
	}

	// Google tokens (future)
	// if strings.HasPrefix(token, "ya29.") {
	// 	return authprovider.ProviderGoogle
	// }

	// Azure tokens are JWTs (future)
	// Note: Azure JWTs start with "eyJ" and are typically > 100 chars
	// Could add: if strings.HasPrefix(token, "eyJ") && len(token) > minJWTLength { return authprovider.ProviderAzure }
	// For now, we fall through to default provider

	// Priority 3: Configuration default
	return s.defaultProvider
}

// extractBearerToken extracts the token from a "Bearer <token>" header value.
func extractBearerToken(authHeader string) (string, error) {
	parts := strings.SplitN(authHeader, " ", authHeaderParts)
	if len(parts) != authHeaderParts {
		return "", errors.New("invalid Authorization header format")
	}

	if !strings.EqualFold(parts[0], "bearer") {
		return "", fmt.Errorf("expected Bearer token, got %s", parts[0])
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("empty token")
	}

	return token, nil
}

// allowResponse creates an OK response with user information headers.
func (s *AuthorizationServer) allowResponse(
	identity *authprovider.UserIdentity,
	orgConstructs []authprovider.OrgConstruct,
) *authv3.CheckResponse {
	headers := []*corev3.HeaderValueOption{
		{
			Header: &corev3.HeaderValue{
				Key:   "x-auth-provider",
				Value: identity.Provider,
			},
		},
		{
			Header: &corev3.HeaderValue{
				Key:   "x-user-id",
				Value: identity.UserID,
			},
		},
		{
			Header: &corev3.HeaderValue{
				Key:   "x-username",
				Value: identity.Username,
			},
		},
	}

	// Add email if available
	if identity.Email != "" {
		headers = append(headers, &corev3.HeaderValueOption{
			Header: &corev3.HeaderValue{
				Key:   "x-user-email",
				Value: identity.Email,
			},
		})
	}

	// Add org constructs
	if len(orgConstructs) > 0 {
		headers = append(headers, &corev3.HeaderValueOption{
			Header: &corev3.HeaderValue{
				Key:   "x-org-constructs",
				Value: strings.Join(extractOrgNames(orgConstructs), ","),
			},
		})
	}

	return &authv3.CheckResponse{
		Status: &status.Status{Code: int32(codes.OK)},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{
				Headers: headers,
			},
		},
	}
}

// denyResponse creates a denial response with the given code and message.
func (s *AuthorizationServer) denyResponse(code codes.Code, message string) *authv3.CheckResponse {
	httpStatus := typev3.StatusCode_Forbidden
	if code == codes.Unauthenticated {
		httpStatus = typev3.StatusCode_Unauthorized
	}

	return &authv3.CheckResponse{
		Status: &status.Status{
			//nolint:gosec // G115: codes.Code is uint32, status.Status.Code is int32. This is safe as gRPC codes are < MaxInt32
			Code:    int32(code),
			Message: message,
		},
		HttpResponse: &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typev3.HttpStatus{
					Code: httpStatus,
				},
				Body: fmt.Sprintf(`{"error": "%s", "message": "%s"}`, code.String(), message),
				Headers: []*corev3.HeaderValueOption{
					{
						Header: &corev3.HeaderValue{
							Key:   "content-type",
							Value: "application/json",
						},
					},
				},
			},
		},
	}
}

// extractOrgNames extracts just the names from org constructs for logging/headers.
func extractOrgNames(orgConstructs []authprovider.OrgConstruct) []string {
	names := make([]string, len(orgConstructs))
	for i, oc := range orgConstructs {
		names[i] = oc.Name
	}

	return names
}

// Validate validates the configuration and checks for common errors.
func (c *Config) Validate() error {
	// Check that roles map is not nil
	if c.Roles == nil {
		c.Roles = make(map[string]Role)
	}

	// Warn about duplicate user assignments across roles
	userAssignments := make(map[string][]string)
	orgAssignments := make(map[string][]string)

	for roleName, role := range c.Roles {
		// Track user assignments
		for _, user := range role.Users {
			userAssignments[strings.ToLower(user)] = append(userAssignments[strings.ToLower(user)], roleName)
		}

		// Track org assignments
		for _, org := range role.Orgs {
			orgAssignments[strings.ToLower(org)] = append(orgAssignments[strings.ToLower(org)], roleName)
		}
	}

	// Warn about conflicts (but don't error - we handle this with "most permissive wins")
	for user, roles := range userAssignments {
		if len(roles) > 1 {
			// Note: In production, you might want to log this as a warning
			// For now, we silently handle it (most permissive wins)
			_ = user // Avoid unused variable warning
		}
	}

	for org, roles := range orgAssignments {
		if len(roles) > 1 {
			// Note: In production, you might want to log this as a warning
			_ = org // Avoid unused variable warning
		}
	}

	// Validate default role exists if specified
	if c.DefaultRole != "" {
		if _, ok := c.Roles[c.DefaultRole]; !ok {
			return fmt.Errorf("default role %q is not defined in roles map", c.DefaultRole)
		}
	}

	return nil
}
