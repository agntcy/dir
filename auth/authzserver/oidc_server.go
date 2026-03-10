// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// Header names for JWT and principal propagation.
const (
	HeaderJWTPayload          = "x-jwt-payload"          // verified JWT payload (set by Envoy jwt_authn)
	HeaderAuthorizedPrincipal = "x-authorized-principal" // canonical principal forwarded to upstream
	HeaderUserID              = "x-user-id"              // user ID (same as principal)
	HeaderPrincipalType       = "x-principal-type"       // principal type (e.g. user, service)
)

// OIDCAuthorizationServer implements the Envoy ext_authz gRPC API for OIDC.
// It reads the verified JWT payload from x-jwt-payload, extracts the principal
// using issuer-specific logic, and enforces Casbin RBAC.
type OIDCAuthorizationServer struct {
	authv3.UnimplementedAuthorizationServer

	config       *OIDCConfig
	roleResolver *OIDCRoleResolver
	logger       *slog.Logger
}

// NewOIDCAuthorizationServer creates a new OIDC-only authorization server.
func NewOIDCAuthorizationServer(config *OIDCConfig, logger *slog.Logger) (*OIDCAuthorizationServer, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if logger == nil {
		logger = slog.Default()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	roleResolver, err := NewOIDCRoleResolver(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create role resolver: %w", err)
	}

	return &OIDCAuthorizationServer{
		config:       config,
		roleResolver: roleResolver,
		logger:       logger,
	}, nil
}

// Check implements the ext_authz Check RPC.
func (s *OIDCAuthorizationServer) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	httpReq := req.GetAttributes().GetRequest().GetHttp()
	path := httpReq.GetPath()

	s.logger.Debug("received authorization request", "path", path, "method", httpReq.GetMethod())

	// 1. Public paths -> Allow
	if s.config.IsPublicPath(path) {
		s.logger.Debug("allowed: public path", "path", path)

		return s.allowResponse("", "public"), nil
	}

	// 2. Read verified JWT payload from x-jwt-payload
	headers := httpReq.GetHeaders()

	payloadJSON := getHeader(headers, HeaderJWTPayload)
	if payloadJSON == "" {
		s.logger.Warn("missing x-jwt-payload header - jwt_authn may not have run or request is unauthenticated")

		return s.denyResponse(codes.Unauthenticated, "missing verified JWT payload"), nil
	}

	// 3. Extract principal (issuer-specific)
	principal, principalType, err := ExtractPrincipal(payloadJSON, s.config)
	if err != nil {
		s.logger.Warn("failed to extract principal", "error", err)

		return s.denyResponse(codes.Unauthenticated, "invalid token: "+err.Error()), nil
	}

	// 4. User deny list -> Deny
	email := GetEmail(payloadJSON, s.config.Claims.EmailPath)
	if s.roleResolver.IsDenied(principal, email) {
		s.logger.Info("denied: principal in deny list", "principal", principal)

		return s.denyResponse(codes.PermissionDenied, "principal is in the deny list"), nil
	}

	// 5. Casbin authorization
	if err := s.roleResolver.Authorize(principal, path); err != nil {
		s.logger.Info("authorization denied", "principal", principal, "path", path, "reason", err.Error())

		return s.denyResponse(codes.PermissionDenied, err.Error()), nil
	}

	// 6. Allow with canonical principal headers
	s.logger.Info("authorization granted", "principal", principal, "path", path)

	return s.allowResponse(principal, principalType), nil
}

func (s *OIDCAuthorizationServer) allowResponse(principal, principalType string) *authv3.CheckResponse {
	headers := []*corev3.HeaderValueOption{}

	if principal != "" {
		headers = append(headers,
			&corev3.HeaderValueOption{
				Header: &corev3.HeaderValue{Key: HeaderAuthorizedPrincipal, Value: principal},
				Append: wrapperspb.Bool(false), // overwrite any client-supplied value
			},
			&corev3.HeaderValueOption{
				Header: &corev3.HeaderValue{Key: HeaderUserID, Value: principal},
				Append: wrapperspb.Bool(false), // overwrite any client-supplied value
			},
		)
		if principalType != "" {
			headers = append(headers, &corev3.HeaderValueOption{
				Header: &corev3.HeaderValue{Key: HeaderPrincipalType, Value: principalType},
				Append: wrapperspb.Bool(false), // overwrite any client-supplied value
			})
		}
	}

	return &authv3.CheckResponse{
		Status: &status.Status{Code: int32(codes.OK)},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: &authv3.OkHttpResponse{Headers: headers},
		},
	}
}

func (s *OIDCAuthorizationServer) denyResponse(code codes.Code, message string) *authv3.CheckResponse {
	httpStatus := typev3.StatusCode_Forbidden
	if code == codes.Unauthenticated {
		httpStatus = typev3.StatusCode_Unauthorized
	}

	return &authv3.CheckResponse{
		Status: &status.Status{
			Code:    int32(code), //nolint:gosec // G115: gRPC codes 0-16 fit in int32; status.Status requires int32
			Message: message,
		},
		HttpResponse: &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typev3.HttpStatus{Code: httpStatus},
				Body:   fmt.Sprintf(`{"error": "%s", "message": "%s"}`, code.String(), message),
				Headers: []*corev3.HeaderValueOption{
					{Header: &corev3.HeaderValue{Key: "content-type", Value: "application/json"}},
				},
			},
		},
	}
}

func getHeader(headers map[string]string, key string) string {
	// Envoy headers may be lowercase
	for k, v := range headers {
		if strings.EqualFold(k, key) {
			return v
		}
	}

	return ""
}
