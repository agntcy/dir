// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"fmt"
	"regexp"
	"slices"
)

// GitHub OIDC issuer URL.
const GitHubIssuer = "https://token.actions.githubusercontent.com"

// Principal type constants for validation.
const (
	PrincipalTypeAuto   = "auto"
	PrincipalTypeUser   = "user"
	PrincipalTypeClient = "client"
	PrincipalTypeGitHub = "github"
)

// OIDCConfig holds the OIDC-based authorization configuration.
// Roles come only from config; no roles are extracted from JWT claims.
type OIDCConfig struct {
	Claims        ClaimsConfig            `yaml:"claims"`
	Issuers       map[string]IssuerConfig `yaml:"issuers"`
	PrincipalType PrincipalTypeConfig     `yaml:"principalType"`
	UserDenyList  []string                `yaml:"userDenyList"`
	PublicPaths   []string                `yaml:"publicPaths"`
	Roles         map[string]OIDCRole     `yaml:"roles"`
}

// ClaimsConfig defines which JWT claims to read.
type ClaimsConfig struct {
	UserID    string `yaml:"userID"`    // e.g. "sub"
	EmailPath string `yaml:"emailPath"` // optional; for userDenyList
}

// IssuerConfig defines issuer-specific principal extraction.
// Allowed principalType: "auto" | "user" | "client" | "github".
type IssuerConfig struct {
	PrincipalType        string `yaml:"principalType"`
	MachineIdentityClaim string `yaml:"machineIdentityClaim"` // e.g. "client_id"
}

// PrincipalTypeConfig is the fallback when issuer is not in Issuers.
// Allowed mode: "auto" | "user" | "client".
type PrincipalTypeConfig struct {
	Mode                 string `yaml:"mode"`
	MachineIdentityClaim string `yaml:"machineIdentityClaim"`
	MachineSubPattern    string `yaml:"machineSubPattern"` // optional regex
}

// OIDCRole defines permissions and principal assignments.
// Principals use user:{iss}:{sub}, client:{iss}:{client_id}, or ghwf:...
type OIDCRole struct {
	AllowedMethods  []string `yaml:"allowedMethods"`
	Users           []string `yaml:"users"`
	Clients         []string `yaml:"clients"`
	GitHubWorkflows []string `yaml:"githubWorkflows"`
}

// Validate validates the OIDC config and returns an error if invalid.
func (c *OIDCConfig) Validate() error {
	if c.Claims.UserID == "" {
		return fmt.Errorf("claims.userID is required")
	}

	// Top-level principalType.mode
	allowedFallbackModes := map[string]bool{
		PrincipalTypeAuto: true, PrincipalTypeUser: true, PrincipalTypeClient: true,
	}
	if c.PrincipalType.Mode != "" && !allowedFallbackModes[c.PrincipalType.Mode] {
		return fmt.Errorf("principalType.mode must be one of [auto, user, client], got %q", c.PrincipalType.Mode)
	}

	// Issuer-specific principalType
	allowedIssuerTypes := map[string]bool{
		PrincipalTypeAuto: true, PrincipalTypeUser: true, PrincipalTypeClient: true, PrincipalTypeGitHub: true,
	}
	for iss, ic := range c.Issuers {
		if ic.PrincipalType != "" && !allowedIssuerTypes[ic.PrincipalType] {
			return fmt.Errorf("issuers[%q].principalType must be one of [auto, user, client, github], got %q", iss, ic.PrincipalType)
		}
	}

	// At least one role
	if len(c.Roles) == 0 {
		return fmt.Errorf("at least one role must be defined")
	}

	for roleName, role := range c.Roles {
		if len(role.AllowedMethods) == 0 {
			return fmt.Errorf("role %q has no allowedMethods", roleName)
		}
	}

	// PublicPaths present (may be empty)
	if c.PublicPaths == nil {
		c.PublicPaths = []string{}
	}

	// Optional: validate machineSubPattern is valid regex if set
	if c.PrincipalType.MachineSubPattern != "" {
		if _, err := regexp.Compile(c.PrincipalType.MachineSubPattern); err != nil {
			return fmt.Errorf("principalType.machineSubPattern is not a valid regex: %w", err)
		}
	}

	return nil
}

// IsPublicPath returns true if the path is in publicPaths (exact match).
func (c *OIDCConfig) IsPublicPath(path string) bool {
	return slices.Contains(c.PublicPaths, path)
}
