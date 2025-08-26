package casbin

import (
	"github.com/casbin/casbin/v2"
	fileadapter "github.com/casbin/casbin/v2/persist/file-adapter"

	"github.com/agntcy/dir/server/authz/casbin/config"
	"github.com/agntcy/dir/server/authz/types"
)

type cbAuthz struct {
	enforcer *casbin.SyncedEnforcer
}

func New(cfg config.Config) (types.Authorizer, error) {
	// Create file adapter
	adapter := fileadapter.NewAdapter(cfg.PolicyPath)

	// Load the Casbin model and policy
	enforcer, err := casbin.NewSyncedEnforcer(cfg.ModelPath, adapter)
	if err != nil {
		return nil, err
	}
	return &cbAuthz{enforcer: enforcer}, nil
}

func (c *cbAuthz) Authorize(userID string, request string) bool {
	allowed, _ := c.enforcer.Enforce(userID, request)
	return allowed
}
