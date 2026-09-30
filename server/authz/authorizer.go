// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	_ "embed"
	"fmt"
	"strings"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authz/config"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// Defines the Casbin authorization model
//
//go:embed model.conf
var modelConf string

// identityOnlyMethods can be granted only by a rule naming the caller's SPIFFE
// ID, never by a rule for a whole trust domain. RequestRegistryCredentials
// hands out the registry password, which reads every record without passing
// through this server's policy gate, so it is for peer nodes, not users.
var identityOnlyMethods = map[string]bool{
	storev1.SyncService_RequestRegistryCredentials_FullMethodName: true,
}

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

	adapter := fileadapter.NewAdapter(cfg.EnforcerPolicyFilePath)

	// Create authorization enforcer
	enforcer, err := casbin.NewEnforcer(model, adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	enforcer.AddFunction("isSpiffeID", stringPredicate("isSpiffeID", func(subject string) bool {
		return strings.HasPrefix(subject, "spiffe://")
	}))
	enforcer.AddFunction("identityOnly", stringPredicate("identityOnly", func(apiMethod string) bool {
		return identityOnlyMethods[apiMethod]
	}))

	return &Authorizer{enforcer: enforcer}, nil
}

// Authorize checks if the caller with the given SPIFFE ID can perform a given
// API method.
//
//nolint:wrapcheck
func (a *Authorizer) Authorize(id spiffeid.ID, apiMethod string) (bool, error) {
	return a.enforcer.Enforce(id.TrustDomain().String(), id.String(), apiMethod)
}

// stringPredicate adapts a predicate on one string to a matcher function.
func stringPredicate(name string, predicate func(string) bool) func(args ...any) (any, error) {
	return func(args ...any) (any, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s: expected 1 argument, got %d", name, len(args))
		}

		value, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("%s: expected a string argument, got %T", name, args[0])
		}

		return predicate(value), nil
	}
}
