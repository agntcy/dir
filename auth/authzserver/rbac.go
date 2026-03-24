// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	_ "embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

//go:embed model.conf
var modelConf string

// OIDCRoleResolver handles role-based authorization using Casbin.
// Principals are user:{iss}:{sub}, client:{iss}:{client_id}, or ghwf:...
// Roles come only from config; no roles are extracted from JWT claims.
type OIDCRoleResolver struct {
	config   *OIDCConfig
	enforcer *casbin.Enforcer
	logger   *slog.Logger
}

// NewOIDCRoleResolver creates a new Casbin-based role resolver for OIDC.
func NewOIDCRoleResolver(config *OIDCConfig, logger *slog.Logger) (*OIDCRoleResolver, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if logger == nil {
		logger = slog.Default()
	}

	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("failed to load Casbin model: %w", err)
	}

	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	resolver := &OIDCRoleResolver{
		config:   config,
		enforcer: enforcer,
		logger:   logger,
	}

	if err := resolver.loadPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return resolver, nil
}

// IsDenied returns true if the principal or oidc:email is in the deny list.
func (r *OIDCRoleResolver) IsDenied(principal, email string) bool {
	if r.isPrincipalDenied(principal) {
		return true
	}

	if email != "" {
		oidcEmail := "oidc:" + email
		if r.isPrincipalDenied(oidcEmail) {
			return true
		}
	}

	return false
}

// Authorize checks if the principal is authorized to access the API method.
// Returns nil if allowed, error if denied.
func (r *OIDCRoleResolver) Authorize(principal, path string) error {
	allowed, err := r.enforcer.Enforce(principal, path, "access")
	if err != nil {
		r.logger.Error("Casbin enforcement error",
			"principal", principal,
			"path", path,
			"error", err,
		)

		return fmt.Errorf("authorization check failed: %w", err)
	}

	if allowed {
		r.logger.Debug("authorized",
			"principal", principal,
			"path", path,
		)

		return nil
	}

	return fmt.Errorf("principal %q is not authorized for %s", principal, path)
}

// isPrincipalDenied checks if the principal is in the deny list.
func (r *OIDCRoleResolver) isPrincipalDenied(principal string) bool {
	for _, denied := range r.config.UserDenyList {
		if strings.EqualFold(principal, denied) {
			return true
		}
	}

	return false
}

// loadPolicies loads Casbin policies from OIDCConfig.
func (r *OIDCRoleResolver) loadPolicies() error {
	var (
		policies  [][]string
		groupings [][]string
	)

	for roleName, role := range r.config.Roles {
		roleKey := "role:" + roleName

		// Permission policies: p, role:X, /path, access
		for _, method := range role.AllowedMethods {
			policies = append(policies, []string{roleKey, method, "access"})
		}

		// Principal-to-role: g, principal, role:X
		for _, u := range role.Users {
			groupings = append(groupings, []string{u, roleKey})
		}

		for _, c := range role.Clients {
			groupings = append(groupings, []string{c, roleKey})
		}

		for _, g := range role.GitHubWorkflows {
			groupings = append(groupings, []string{g, roleKey})
		}
	}

	if len(policies) > 0 {
		if _, err := r.enforcer.AddPolicies(policies); err != nil {
			return fmt.Errorf("failed to add permission policies: %w", err)
		}
	}

	if len(groupings) > 0 {
		if _, err := r.enforcer.AddGroupingPolicies(groupings); err != nil {
			return fmt.Errorf("failed to add principal-role mappings: %w", err)
		}
	}

	r.logger.Info("Casbin policies loaded",
		"permissions", len(policies),
		"groupings", len(groupings),
	)

	return nil
}
