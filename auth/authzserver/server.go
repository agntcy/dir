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
	config          *Config
	logger          *slog.Logger
	defaultProvider string
}

// Config holds the authorization server configuration.
type Config struct {
	// DefaultProvider is used when provider cannot be auto-detected
	DefaultProvider string

	// AllowedOrgConstructs restricts access to users in these org constructs.
	// Works across all providers (GitHub orgs, Azure tenants, Google domains, etc.)
	// Empty list means no org restriction.
	AllowedOrgConstructs []string

	// UserAllowList explicitly allows specific users regardless of org membership.
	// Format: "provider:username" (e.g., "github:tkircsi", "google:alice@agntcy.com")
	UserAllowList []string

	// UserDenyList explicitly denies specific users (takes precedence over allow lists).
	// Format: "provider:username"
	UserDenyList []string
}

// NewAuthorizationServer creates a new authorization server with multiple providers.
func NewAuthorizationServer(
	providers map[string]authprovider.Provider,
	config *Config,
	logger *slog.Logger,
) *AuthorizationServer {
	if config == nil {
		config = &Config{}
	}

	if logger == nil {
		logger = slog.Default()
	}

	if config.DefaultProvider == "" {
		config.DefaultProvider = authprovider.ProviderGithub
	}

	return &AuthorizationServer{
		providers:       providers,
		config:          config,
		logger:          logger,
		defaultProvider: config.DefaultProvider,
	}
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

	// Check authorization rules
	if err := s.checkAuthorization(identity, orgConstructs); err != nil {
		s.logger.Info("authorization denied",
			"provider", identity.Provider,
			"user", identity.Username,
			"org_constructs", extractOrgNames(orgConstructs),
			"reason", err.Error(),
		)

		return s.denyResponse(codes.PermissionDenied, err.Error()), nil
	}

	s.logger.Info("authorization granted",
		"provider", identity.Provider,
		"user", identity.Username,
		"org_constructs", extractOrgNames(orgConstructs),
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

// checkAuthorization checks if the user is authorized based on configured rules.
func (s *AuthorizationServer) checkAuthorization(
	identity *authprovider.UserIdentity,
	orgConstructs []authprovider.OrgConstruct,
) error {
	userKey := fmt.Sprintf("%s:%s", identity.Provider, identity.Username)

	// Priority 1: Check deny list (highest priority)
	for _, denied := range s.config.UserDenyList {
		if strings.EqualFold(userKey, denied) || strings.EqualFold(identity.Username, denied) {
			return fmt.Errorf("user %q is in the deny list", identity.Username)
		}
	}

	// Priority 2: Check user allow list (explicit allow)
	for _, allowed := range s.config.UserAllowList {
		if strings.EqualFold(userKey, allowed) || strings.EqualFold(identity.Username, allowed) {
			return nil // Explicitly allowed, skip other checks
		}
	}

	// Priority 3: Check org construct membership
	if len(s.config.AllowedOrgConstructs) == 0 {
		return nil // No org restrictions, allow all authenticated users
	}

	// Build map of user's org constructs
	userOrgSet := make(map[string]bool)
	for _, oc := range orgConstructs {
		userOrgSet[strings.ToLower(oc.Name)] = true
	}

	// Check against allowed list
	for _, allowed := range s.config.AllowedOrgConstructs {
		if userOrgSet[strings.ToLower(allowed)] {
			return nil // User in allowed org construct
		}
	}

	return fmt.Errorf("user %q is not a member of any allowed organization/tenant/domain", identity.Username)
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
