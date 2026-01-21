// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	_ "embed"
	"fmt"

	"github.com/agntcy/dir/server/authz/config"
	"github.com/casbin/casbin/v2"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
)

type Authorizer struct {
	enforcer *casbin.Enforcer
}

// New creates a new Casbin-based Authorizer.
func NewAuthorizer(cfg config.Config) (*Authorizer, error) {

	adapter := fileadapter.NewAdapter(cfg.EnforcerPolicyFilePath)

	// Create authorization enforcer
	enforcer, err := casbin.NewEnforcer(cfg.EnforcerModelFilePath, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	return &Authorizer{enforcer: enforcer}, nil
}

// Authorize checks if the user in trust domain can perform a given API method.
//
//nolint:wrapcheck
func (a *Authorizer) Authorize(trustDomain, apiMethod string) (bool, error) {
	return a.enforcer.Enforce(trustDomain, apiMethod)
}
