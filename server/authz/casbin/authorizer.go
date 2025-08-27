package casbin

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authz/casbin/config"
	"github.com/casbin/casbin/v2"
)

type Authorizer struct {
	enforcer *casbin.Enforcer
}

// New creates a new Casbin Authorizer
func New(cfg config.Config) (*Authorizer, error) {
	e, err := casbin.NewEnforcer(cfg.ModelPath, cfg.PolicyDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	return &Authorizer{enforcer: e}, nil
}

// Authorize checks if the user_id can perform api_method in trust_domain.
func (a *Authorizer) Authorize(ctx context.Context, userID, apiMethod, trustDomain string) (bool, error) {
	return a.enforcer.Enforce(userID, apiMethod, trustDomain)
}
