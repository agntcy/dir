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

// RoleResolver handles role-based authorization decisions using Casbin.
// It supports user-to-role and org-to-role mappings with flexible policy definitions.
type RoleResolver struct {
	config   *Config
	enforcer *casbin.Enforcer
	logger   *slog.Logger
}

// NewRoleResolver creates a new Casbin-based role resolver.
func NewRoleResolver(config *Config, logger *slog.Logger) (*RoleResolver, error) {
	if config == nil {
		config = &Config{}
	}

	if logger == nil {
		logger = slog.Default()
	}

	// Create Casbin model from embedded string
	m, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("failed to load Casbin model: %w", err)
	}

	// Create Casbin enforcer
	enforcer, err := casbin.NewEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create Casbin enforcer: %w", err)
	}

	resolver := &RoleResolver{
		config:   config,
		enforcer: enforcer,
		logger:   logger,
	}

	// Load policies from config
	if err := resolver.loadPolicies(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return resolver, nil
}

// Authorize checks if a user is authorized to access the requested API method.
// Priority:
// 1. Deny list (blocks all access)
// 2. User-specific role assignment
// 3. Organization-based role assignment
// 4. Default role
// 5. Deny by default.
func (r *RoleResolver) Authorize(username, userKey string, userOrgs []string, apiMethod string) error {
	// Priority 1: Check deny list (highest priority - blocks all access)
	if r.isUserDenied(userKey, username) {
		return fmt.Errorf("user %q is in the deny list", username)
	}

	// Priority 2: Check user permissions directly (user-to-role mapping)
	if err := r.checkUserRole(username, userKey, apiMethod); err == nil {
		return nil // Authorized
	}

	// Priority 3: Check org permissions (org-to-role mapping)
	if err := r.checkOrgRoles(username, userOrgs, apiMethod); err == nil {
		return nil // Authorized
	}

	// Priority 4: Check default role
	return r.checkDefaultRole(username, apiMethod)
}

// checkUserRole checks if the user has direct role assignment that allows the method.
func (r *RoleResolver) checkUserRole(username, userKey, apiMethod string) error {
	allowed, err := r.enforcer.Enforce(userKey, apiMethod, "access")
	if err != nil {
		r.logger.Error("Casbin enforcement error for user",
			"user", username,
			"method", apiMethod,
			"error", err,
		)

		return fmt.Errorf("authorization check failed: %w", err)
	}

	if allowed {
		r.logger.Debug("method authorized via user role",
			"user", username,
			"method", apiMethod,
		)

		return nil // Authorized
	}

	return fmt.Errorf("user role check failed") // Not authorized, but not an error
}

// checkOrgRoles checks if any of the user's organizations have a role that allows the method.
func (r *RoleResolver) checkOrgRoles(username string, userOrgs []string, apiMethod string) error {
	for _, org := range userOrgs {
		allowed, err := r.enforcer.Enforce("org:"+org, apiMethod, "access")
		if err != nil {
			r.logger.Error("Casbin enforcement error for org",
				"org", org,
				"method", apiMethod,
				"error", err,
			)

			continue // Skip this org, try next
		}

		if allowed {
			r.logger.Debug("method authorized via org role",
				"user", username,
				"org", org,
				"method", apiMethod,
			)

			return nil // Authorized
		}
	}

	return fmt.Errorf("org role check failed") // Not authorized
}

// checkDefaultRole checks if the default role allows the method.
func (r *RoleResolver) checkDefaultRole(username, apiMethod string) error {
	if r.config.DefaultRole == "" {
		// No default role configured - deny by default
		return fmt.Errorf("user %q has no assigned role and no default role is configured", username)
	}

	allowed, err := r.enforcer.Enforce("role:"+r.config.DefaultRole, apiMethod, "access")
	if err != nil {
		r.logger.Error("Casbin enforcement error for default role",
			"role", r.config.DefaultRole,
			"method", apiMethod,
			"error", err,
		)

		return fmt.Errorf("authorization check failed: %w", err)
	}

	if allowed {
		r.logger.Debug("method authorized via default role",
			"user", username,
			"role", r.config.DefaultRole,
			"method", apiMethod,
		)

		return nil // Authorized
	}

	return fmt.Errorf("user %q with default role %q is not authorized for method %s",
		username, r.config.DefaultRole, apiMethod)
}

// isUserDenied checks if the user is in the deny list.
func (r *RoleResolver) isUserDenied(userKey, username string) bool {
	for _, denied := range r.config.UserDenyList {
		if strings.EqualFold(userKey, denied) || strings.EqualFold(username, denied) {
			return true
		}
	}

	return false
}

// loadPolicies converts our Config into Casbin policies and loads them into the enforcer.
func (r *RoleResolver) loadPolicies() error {
	var (
		policies         [][]string
		userRoleMappings [][]string
		orgRoleMappings  [][]string
	)

	// Convert roles to Casbin policies

	for roleName, role := range r.config.Roles {
		roleKey := "role:" + roleName

		// Add permission policies for this role
		for _, method := range role.AllowedMethods {
			// p, role:admin, /store.StoreService/*, access
			policies = append(policies, []string{roleKey, method, "access"})
		}

		// Add user-to-role mappings
		for _, user := range role.Users {
			// Normalize user key: support both "github:alice" and "alice"
			userKey := user
			if !strings.Contains(user, ":") {
				userKey = "github:" + user
			}
			// g, github:alice, role:admin
			userRoleMappings = append(userRoleMappings, []string{userKey, roleKey})
		}

		// Add org-to-role mappings
		for _, org := range role.Orgs {
			// g2, org:agntcy, role:admin
			orgRoleMappings = append(orgRoleMappings, []string{"org:" + org, roleKey})
		}
	}

	// Load all policies
	if len(policies) > 0 {
		if _, err := r.enforcer.AddPolicies(policies); err != nil {
			return fmt.Errorf("failed to add permission policies: %w", err)
		}
	}

	if len(userRoleMappings) > 0 {
		if _, err := r.enforcer.AddGroupingPolicies(userRoleMappings); err != nil {
			return fmt.Errorf("failed to add user-role mappings: %w", err)
		}
	}

	if len(orgRoleMappings) > 0 {
		if _, err := r.enforcer.AddNamedGroupingPolicies("g2", orgRoleMappings); err != nil {
			return fmt.Errorf("failed to add org-role mappings: %w", err)
		}
	}

	r.logger.Info("Casbin policies loaded",
		"permissions", len(policies),
		"user_roles", len(userRoleMappings),
		"org_roles", len(orgRoleMappings),
	)

	return nil
}
