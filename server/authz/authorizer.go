// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	_ "embed"
	"fmt"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authz/config"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

// Defines the Casbin authorization model
//
//go:embed model.conf
var modelConf string

type Authorizer struct {
	enforcer *casbin.Enforcer
}

// New creates a new Casbin-based Authorizer.
func NewAuthorizer(cfg config.Config) (*Authorizer, error) {
	// Create model from string
	model, err := model.NewModelFromString(modelConf)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Create authorization enforcer
	enforcer, err := casbin.NewEnforcer(model)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	// Add policies to the enforcer
	if _, err := enforcer.AddPolicies(getPolicies(cfg)); err != nil {
		return nil, fmt.Errorf("failed to add policies: %w", err)
	}

	return &Authorizer{enforcer: enforcer}, nil
}

// Authorize checks if the user in trust domain can perform a given API method.
//
//nolint:wrapcheck
func (a *Authorizer) Authorize(trustDomain, apiMethod string) (bool, error) {
	return a.enforcer.Enforce(trustDomain, apiMethod)
}

// getPolicies returns a list of authorization in the following form:
func getPolicies(cfg config.Config) [][]string {
	var DefaultPolicies = [][]string{
		{cfg.TrustDomain, "*"},
		{"*", storev1.StoreService_Pull_FullMethodName},
	}

	if len(cfg.Policies) == 0 {
		logger.Info("No user-defined policies found, using default policies.")

		return DefaultPolicies
	}

	policies := [][]string{}

	// Apply user-defined policies
	for domain, methods := range cfg.Policies {
		for _, method := range methods {
			policies = append(policies, []string{domain, method})
		}
	}

	return policies
}
