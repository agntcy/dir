package casbin

import (
	"context"
	_ "embed"
	"fmt"

	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/server/authz/config"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

//go:embed model.conf
var modelConf string

var allowedExternalAPIMethods = []string{
	storev1.StoreService_Pull_FullMethodName,                      // store: pull
	storev1.StoreService_PullReferrer_FullMethodName,              // store: pull referrer
	storev1.StoreService_Lookup_FullMethodName,                    // store: lookup
	storev1.SyncService_RequestRegistryCredentials_FullMethodName, // sync: negotiate
}

type Authorizer struct {
	enforcer *casbin.Enforcer
}

// New creates a new Casbin Authorizer
func New(cfg config.Config) (*Authorizer, error) {
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
func (a *Authorizer) Authorize(ctx context.Context, trustDomain, apiMethod string) (bool, error) {
	return a.enforcer.Enforce(trustDomain, apiMethod)
}

func getPolicies(cfg config.Config) [][]string {
	var policies [][]string

	// Allow all API methods for the trust domain
	policies = append(policies, []string{cfg.TrustDomain, "*"})

	// Allow only specific API methods for users outside of the trust domain
	for _, method := range allowedExternalAPIMethods {
		policies = append(policies, []string{"*", method})
	}

	return policies
}
