// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authzserver

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ExtractPrincipal extracts the canonical principal from the JWT payload JSON.
// Uses issuer-specific logic: GitHub -> ghwf:..., else -> user:{iss}:{sub} or client:{iss}:{client_id}.
//
//nolint:nonamedreturns // named returns clarify principal, principalType, err for callers
func ExtractPrincipal(payloadJSON string, config *OIDCConfig) (principal string, principalType string, err error) {
	if config == nil {
		return "", "", fmt.Errorf("config is required")
	}

	if payloadJSON == "" {
		return "", "", fmt.Errorf("empty JWT payload")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return "", "", fmt.Errorf("invalid JWT payload JSON: %w", err)
	}

	iss := getString(payload, "iss")
	if iss == "" {
		return "", "", fmt.Errorf("missing iss claim")
	}

	// Issuer-specific extraction
	if ic := config.GetIssuerConfig(iss); ic != nil {
		machineSub := ic.MachineSubPattern
		if machineSub == "" {
			machineSub = config.PrincipalType.MachineSubPattern
		}

		switch ic.PrincipalType {
		case PrincipalTypeGitHub:
			return extractGitHubPrincipal(payload)
		case PrincipalTypeUser:
			return extractUserPrincipal(payload, iss)
		case PrincipalTypeClient:
			return extractClientPrincipal(payload, iss, ic.MachineIdentityClaim)
		case PrincipalTypeAuto, "":
			return extractAutoPrincipal(payload, iss, ic.MachineIdentityClaim, machineSub)
		}
	}

	// Fallback to top-level principalType
	pt := config.PrincipalType
	switch pt.Mode {
	case PrincipalTypeUser:
		return extractUserPrincipal(payload, iss)
	case PrincipalTypeClient:
		return extractClientPrincipal(payload, iss, pt.MachineIdentityClaim)
	case PrincipalTypeAuto, "":
		return extractAutoPrincipal(payload, iss, pt.MachineIdentityClaim, pt.MachineSubPattern)
	}

	return "", "", fmt.Errorf("unknown principal type mode: %s", pt.Mode)
}

func extractUserPrincipal(payload map[string]any, iss string) (string, string, error) {
	sub := getString(payload, "sub")
	if sub == "" {
		return "", "", fmt.Errorf("missing sub claim for user principal")
	}

	return fmt.Sprintf("user:%s:%s", iss, sub), PrincipalTypeUser, nil
}

func extractClientPrincipal(payload map[string]any, iss, machineClaim string) (string, string, error) {
	clientID := getMachineIdentity(payload, machineClaim)
	if clientID == "" {
		return "", "", fmt.Errorf("missing %s claim for client principal", machineClaim)
	}

	return fmt.Sprintf("client:%s:%s", iss, clientID), PrincipalTypeClient, nil
}

func extractAutoPrincipal(payload map[string]any, iss, machineClaim, machineSubPattern string) (string, string, error) {
	sub := getString(payload, "sub")
	clientID := getMachineIdentity(payload, machineClaim)

	// No sub -> machine
	if sub == "" {
		if clientID == "" {
			return "", "", fmt.Errorf("missing sub and %s for principal", machineClaim)
		}

		return fmt.Sprintf("client:%s:%s", iss, clientID), PrincipalTypeClient, nil
	}

	// machineSubPattern matches sub -> machine
	if machineSubPattern != "" {
		re, err := regexp.Compile(machineSubPattern)
		if err == nil && re.MatchString(sub) {
			if clientID == "" {
				return "", "", fmt.Errorf("sub matches machine pattern but missing %s", machineClaim)
			}

			return fmt.Sprintf("client:%s:%s", iss, clientID), PrincipalTypeClient, nil
		}
	}

	// sub == client_id -> machine
	if clientID != "" && sub == clientID {
		return fmt.Sprintf("client:%s:%s", iss, clientID), PrincipalTypeClient, nil
	}

	// Default: user
	return fmt.Sprintf("user:%s:%s", iss, sub), PrincipalTypeUser, nil
}

func extractGitHubPrincipal(payload map[string]any) (string, string, error) {
	repo := getString(payload, "repository")
	ref := getString(payload, "ref")
	env := getString(payload, "environment")

	// Prefer workflow_ref, fallback to job_workflow_ref
	workflowRef := getString(payload, "workflow_ref")
	if workflowRef == "" {
		workflowRef = getString(payload, "job_workflow_ref")
	}

	if repo == "" {
		return "", "", fmt.Errorf("missing repository claim for GitHub principal")
	}

	// Extract workflow file from workflow_ref: owner/repo/.github/workflows/file.yml@ref
	workflowFile := ""

	if workflowRef != "" {
		// Format: owner/repo/.github/workflows/deploy.yml@refs/heads/main
		parts := strings.Split(workflowRef, "@")
		if len(parts) >= 1 {
			path := parts[0]
			if idx := strings.Index(path, ".github/workflows/"); idx >= 0 {
				workflowFile = strings.TrimPrefix(path[idx:], ".github/workflows/")
			}
		}
	}

	if workflowFile == "" {
		return "", "", fmt.Errorf("missing workflow_ref or job_workflow_ref for GitHub principal")
	}

	if ref == "" {
		ref = "refs/heads/main" // fallback
	}

	principal := fmt.Sprintf("ghwf:repo:%s:workflow:%s:ref:%s", repo, workflowFile, ref)
	if env != "" {
		principal += ":env:" + env
	}

	return principal, "ghwf", nil
}

func getString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}

	s, _ := v.(string)

	return s
}

func getMachineIdentity(payload map[string]any, claim string) string {
	if claim == "" {
		claim = "client_id"
	}

	s := getString(payload, claim)
	if s != "" {
		return s
	}

	return getString(payload, "azp")
}

// GetEmail extracts email from payload for deny list matching.
// Supports dot notation for nested paths (e.g. "email" or "claims.email").
func GetEmail(payloadJSON, emailPath string) string {
	if payloadJSON == "" || emailPath == "" {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return ""
	}

	v := getNestedValue(payload, strings.Split(emailPath, "."))
	if s, ok := v.(string); ok {
		return s
	}

	return ""
}

func getNestedValue(m map[string]any, path []string) any {
	if len(path) == 0 {
		return nil
	}

	v, ok := m[path[0]]
	if !ok {
		return nil
	}

	if len(path) == 1 {
		return v
	}

	next, ok := v.(map[string]any)
	if !ok {
		return nil
	}

	return getNestedValue(next, path[1:])
}
