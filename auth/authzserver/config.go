// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
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
	Claims        ClaimsConfig        `yaml:"claims"`
	Issuers       []IssuerConfig      `yaml:"issuers"`
	PrincipalType PrincipalTypeConfig `yaml:"principalType"`
	UserDenyList  []string            `yaml:"userDenyList"`
	PublicPaths   []string            `yaml:"publicPaths"`
	Roles         map[string]OIDCRole `yaml:"roles"`
}

// ClaimsConfig defines which JWT claims to read.
type ClaimsConfig struct {
	UserID    string `yaml:"userID"`    // e.g. "sub"
	EmailPath string `yaml:"emailPath"` // optional; for userDenyList
}

// IssuerConfig defines issuer-specific principal extraction.
// Provider is the OIDC issuer URL (e.g. https://tenant.zitadel.cloud).
// Allowed principalType: "auto" | "user" | "client" | "github".
type IssuerConfig struct {
	Provider             string `yaml:"provider"`
	PrincipalType        string `yaml:"principalType"`
	MachineIdentityClaim string `yaml:"machineIdentityClaim"` // e.g. "client_id"
	MachineSubPattern    string `yaml:"machineSubPattern"`    // optional regex for auto mode
}

// GetIssuerConfig returns the IssuerConfig for the given issuer URL, or nil if not found.
func (c *OIDCConfig) GetIssuerConfig(issuerURL string) *IssuerConfig {
	for i := range c.Issuers {
		if c.Issuers[i].Provider == issuerURL {
			return &c.Issuers[i]
		}
	}

	return nil
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
	AllowedMethods []string `yaml:"allowedMethods"`
	Users          []string `yaml:"users"`
	Clients        []string `yaml:"clients"`
	// GitHubWorkflows supports exact principals and optional '*' wildcard
	// for ghwf principals only (e.g. ...:ref:refs/heads/*).
	GitHubWorkflows []string `yaml:"githubWorkflows"`
}

// Validate validates the OIDC config and returns an error if invalid.
func (c *OIDCConfig) Validate() error {
	if err := c.validateClaims(); err != nil {
		return err
	}

	if err := c.validatePrincipalType(); err != nil {
		return err
	}

	if err := c.validateIssuers(); err != nil {
		return err
	}

	if err := c.validateRoles(); err != nil {
		return err
	}

	c.normalizePublicPaths()

	return nil
}

func (c *OIDCConfig) validateClaims() error {
	if c.Claims.UserID == "" {
		return fmt.Errorf("claims.userID is required")
	}

	return nil
}

var allowedFallbackModes = map[string]bool{
	PrincipalTypeAuto: true, PrincipalTypeUser: true, PrincipalTypeClient: true,
}

var allowedIssuerTypes = map[string]bool{
	PrincipalTypeAuto: true, PrincipalTypeUser: true, PrincipalTypeClient: true, PrincipalTypeGitHub: true,
}

func (c *OIDCConfig) validatePrincipalType() error {
	if c.PrincipalType.Mode != "" && !allowedFallbackModes[c.PrincipalType.Mode] {
		return fmt.Errorf("principalType.mode must be one of [auto, user, client], got %q", c.PrincipalType.Mode)
	}

	if c.PrincipalType.MachineSubPattern != "" {
		if _, err := regexp.Compile(c.PrincipalType.MachineSubPattern); err != nil {
			return fmt.Errorf("principalType.machineSubPattern is not a valid regex: %w", err)
		}
	}

	return nil
}

func (c *OIDCConfig) validateIssuers() error {
	for i, ic := range c.Issuers {
		if ic.Provider == "" {
			return fmt.Errorf("issuers[%d].provider is required", i)
		}

		if ic.PrincipalType != "" && !allowedIssuerTypes[ic.PrincipalType] {
			return fmt.Errorf("issuers[%q].principalType must be one of [auto, user, client, github], got %q", ic.Provider, ic.PrincipalType)
		}

		if ic.MachineSubPattern != "" {
			if _, err := regexp.Compile(ic.MachineSubPattern); err != nil {
				return fmt.Errorf("issuers[%q].machineSubPattern is not a valid regex: %w", ic.Provider, err)
			}
		}
	}

	return nil
}

func (c *OIDCConfig) validateRoles() error {
	if len(c.Roles) == 0 {
		return fmt.Errorf("at least one role must be defined")
	}

	for roleName, role := range c.Roles {
		if len(role.AllowedMethods) == 0 {
			return fmt.Errorf("role %q has no allowedMethods", roleName)
		}

		for _, workflowPrincipal := range role.GitHubWorkflows {
			if strings.Contains(workflowPrincipal, "*") && !isSupportedGitHubWorkflowWildcard(workflowPrincipal) {
				return fmt.Errorf(
					"role %q has invalid githubWorkflows wildcard %q: only one '*' is supported, it must be at the end, and only in ghwf ...:ref:refs/heads/<branch>*",
					roleName,
					workflowPrincipal,
				)
			}
		}
	}

	return nil
}

// isSupportedGitHubWorkflowWildcard enforces a constrained wildcard format:
// - principal must start with ghwf:repo:
// - exactly one '*' is allowed
// - '*' must be the final character
// - wildcard must be inside :ref:refs/heads/<branch>* segment.
func isSupportedGitHubWorkflowWildcard(principal string) bool {
	const branchRefPrefix = ":ref:refs/heads/"

	if !strings.HasPrefix(principal, "ghwf:repo:") {
		return false
	}

	if strings.Count(principal, "*") != 1 {
		return false
	}

	if !strings.HasSuffix(principal, "*") {
		return false
	}

	refIdx := strings.Index(principal, branchRefPrefix)
	if refIdx < 0 {
		return false
	}

	starPos := len(principal) - 1
	minBranchStart := refIdx + len(branchRefPrefix)

	// The wildcard must be in the branch segment.
	if starPos < minBranchStart {
		return false
	}

	return true
}

func (c *OIDCConfig) normalizePublicPaths() {
	if c.PublicPaths == nil {
		c.PublicPaths = []string{}
	}
}

// IsPublicPath returns true if the path is in publicPaths (exact match).
func (c *OIDCConfig) IsPublicPath(path string) bool {
	return slices.Contains(c.PublicPaths, path)
}
