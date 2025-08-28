// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package opa

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authz/opa/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/open-policy-agent/opa/v1/rego"
)

const authzQuery = "data.authz.allow"

var logger = logging.Logger("authz/opa")

type Authorizer struct {
	query *rego.PreparedEvalQuery
}

func New(ctx context.Context, cfg config.Config) (*Authorizer, error) {
	// Create a new evaluation query
	query, err := rego.New(
		rego.Query(authzQuery),
		rego.LoadBundle(cfg.BundlePath),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create evaluation query: %w", err)
	}

	return &Authorizer{query: &query}, nil
}

func (c *Authorizer) Authorize(ctx context.Context, trustDomain, userID, apiMethod string) (bool, error) {
	results, err := c.query.Eval(ctx, rego.EvalInput(map[string]interface{}{
		"api_method":   apiMethod,
		"user_id":      userID,
		"trust_domain": trustDomain,
	}))
	if err != nil {
		return false, fmt.Errorf("failed to evaluate query: %w", err)
	}

	return results.Allowed(), nil
}
