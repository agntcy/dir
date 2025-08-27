// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package opa

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authz/opa/config"
	"github.com/agntcy/dir/server/authz/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/open-policy-agent/opa/v1/rego"
)

const authzQuery = "data.authz.allow"

var logger = logging.Logger("authz/opa")

type opaAuthz struct {
	query rego.PreparedEvalQuery
}

func New(ctx context.Context, cfg config.Config) (types.Authorizer, error) {
	query, err := rego.New(
		rego.Query(authzQuery),
		rego.Load([]string{cfg.PolicyDirPath}, nil),
	).PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA query: %w", err)
	}

	return &opaAuthz{query: query}, nil
}

func (c *opaAuthz) Authorize(ctx context.Context, userID string, request string) bool {
	results, err := c.query.Eval(ctx, rego.EvalInput(map[string]interface{}{
		"request": request,
		"user_id": userID,
	}))
	if err != nil {
		logger.Error("OPA failed to evaluate query", "error", err)

		return false
	}

	return results.Allowed()
}
