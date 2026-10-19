// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package publish implements the publish reconciler task.
// It searches the local index for records matching Criteria and submits
// publication requests for the publication worker. A run pages through the
// match set, then publishes.
package publish

import (
	"context"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("reconciler/publish")

// store is the search-index and publication-queue surface this task needs.
type store interface {
	GetRecordCIDs(opts ...types.FilterOption) ([]string, error)
	CreatePublication(request *routingv1.PublishRequest) (string, error)
}

// Task implements the publish reconciler task.
type Task struct {
	config Config
	db     store
}

// NewTask creates a new publish task.
func NewTask(config Config, db store) (*Task, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Task{
		config: config,
		db:     db,
	}, nil
}

// Name returns the task name.
func (t *Task) Name() string {
	return "publish"
}

// Interval returns how often this task should run.
func (t *Task) Interval() time.Duration {
	return t.config.GetInterval()
}

// IsEnabled returns whether this task is enabled.
func (t *Task) IsEnabled() bool {
	return t.config.Enabled
}

// Run pages through local records matching Criteria and submits one
// publication request per page. Publishing does not remove a record from
// the match set, so a single run walks --offset until a short page.
func (t *Task) Run(ctx context.Context) error {
	domains := t.config.GetDomains()
	skills := t.config.GetSkills()
	limit := t.config.GetLimit()

	logger.Debug("Running publish", "domains", domains, "skills", skills, "limit", limit)

	var total, pages int

	for offset := 0; ; offset += limit {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("publish canceled: %w", err)
		}

		cids, err := t.page(offset, limit, domains, skills)
		if err != nil {
			return err
		}

		if len(cids) == 0 {
			break
		}

		pages++
		total += len(cids)

		if t.config.DryRun {
			logger.Info("Dry run: not creating publication",
				"offset", offset,
				"count", len(cids),
				"domains", domains,
				"skills", skills,
			)
			logger.Debug("Dry run: matching CIDs", "offset", offset, "cids", cids)
		} else if err := t.publishPage(cids); err != nil {
			return err
		}

		if len(cids) < limit {
			break
		}
	}

	if total == 0 {
		logger.Info("No records to publish", "domains", domains, "skills", skills)

		return nil
	}

	if t.config.DryRun {
		logger.Info("Dry run: publish complete", "records", total, "pages", pages, "domains", domains, "skills", skills)

		return nil
	}

	logger.Info("Publish complete", "records", total, "pages", pages, "domains", domains, "skills", skills)

	return nil
}

func (t *Task) page(offset, limit int, domains, skills []string) ([]string, error) {
	opts := []types.FilterOption{
		types.WithLimit(limit),
		types.WithOffset(offset),
	}
	if len(domains) > 0 {
		opts = append(opts, types.WithDomainNames(domains...))
	}

	if len(skills) > 0 {
		opts = append(opts, types.WithSkillNames(skills...))
	}

	cids, err := t.db.GetRecordCIDs(opts...)
	if err != nil {
		return nil, fmt.Errorf("get records: %w", err)
	}

	out := make([]string, 0, len(cids))
	for _, cid := range cids {
		if cid != "" {
			out = append(out, cid)
		}
	}

	return out, nil
}

func (t *Task) publishPage(cids []string) error {
	refs := make([]*corev1.RecordRef, 0, len(cids))
	for _, cid := range cids {
		refs = append(refs, &corev1.RecordRef{Cid: cid})
	}

	id, err := t.db.CreatePublication(&routingv1.PublishRequest{
		Request: &routingv1.PublishRequest_RecordRefs{
			RecordRefs: &routingv1.RecordRefs{Refs: refs},
		},
	})
	if err != nil {
		return fmt.Errorf("create publication: %w", err)
	}

	logger.Info("Publication created", "publication_id", id, "cids", len(cids))

	return nil
}
