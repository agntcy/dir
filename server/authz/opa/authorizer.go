// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package opa

import (
	"context"
	"fmt"
	"sync"

	"github.com/agntcy/dir/server/authz/opa/config"
	"github.com/agntcy/dir/server/authz/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/fsnotify/fsnotify"
	"github.com/open-policy-agent/opa/v1/rego"
)

const authzQuery = "data.authz.allow"

var logger = logging.Logger("authz/opa")

type opaAuthz struct {
	mu    sync.Mutex
	query *rego.PreparedEvalQuery
	close func() error
}

func New(ctx context.Context, cfg config.Config) (types.Authorizer, error) {
	opa := &opaAuthz{}
	if err := opa.init(ctx, cfg.BundlePath); err != nil {
		return nil, fmt.Errorf("failed to initialize authorizer: %w", err)
	}

	return opa, nil
}

func (c *opaAuthz) Authorize(ctx context.Context, trustDomain, userID, apiMethod string) bool {
	results, err := c.query.Eval(ctx, rego.EvalInput(map[string]interface{}{
		"api_method":   apiMethod,
		"user_id":      userID,
		"trust_domain": trustDomain,
	}))
	if err != nil {
		logger.Error("failed to evaluate query", "error", err)

		return false
	}

	return results.Allowed()
}

func (c *opaAuthz) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.close != nil {
		_ = c.close()
		c.close = nil
	}
}

func (c *opaAuthz) init(ctx context.Context, bundlePath string) error {
	// Reload to fetch current policies
	if err := c.reload(ctx, bundlePath); err != nil {
		return fmt.Errorf("failed to fetch policies: %w", err)
	}

	// Create a policy dir watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Start listening for policy changes
	//nolint:contextcheck
	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}

				if err := c.reload(context.Background(), bundlePath); err != nil {
					logger.Error("failed to reload policies", "error", err)
				}

				logger.Info("policies reloaded")

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}

				logger.Error("file watcher error", "error", err)
			}
		}
	}()

	// Add policy path to watcher
	err = watcher.Add(bundlePath)
	if err != nil {
		return fmt.Errorf("failed to add watcher: %w", err)
	}

	// Set closer
	c.close = watcher.Close

	return nil
}

func (c *opaAuthz) reload(ctx context.Context, bundlePath string) error {
	query, err := rego.New(
		rego.Query(authzQuery),
		rego.LoadBundle(bundlePath),
	).PrepareForEval(ctx)
	if err != nil {
		return fmt.Errorf("failed to create evaluation query: %w", err)
	}

	c.mu.Lock()
	c.query = &query
	c.mu.Unlock()

	return nil
}
