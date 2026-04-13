// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package fetcher

import (
	"context"
	"errors"
	"strings"

	"github.com/agntcy/dir/importer/skill"
	"github.com/agntcy/dir/importer/types"
)

// agentSkillDirFetcher reads one Agent Skill directory (SKILL.md).
type agentSkillDirFetcher struct {
	path string
}

// NewAgentSkillDirFetcher creates a fetcher that reads a single skill from a directory path.
func NewAgentSkillDirFetcher(path string) (*agentSkillDirFetcher, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("skill directory path is empty")
	}

	return &agentSkillDirFetcher{path: path}, nil
}

// Fetch parses the skill directory and emits one SourceItem.
func (f *agentSkillDirFetcher) Fetch(ctx context.Context) (<-chan types.SourceItem, <-chan error) {
	const chanBuf = 8

	outputCh := make(chan types.SourceItem, chanBuf)
	errCh := make(chan error, 1)

	go func() {
		defer close(outputCh)
		defer close(errCh)

		st, err := skill.ParseSkillDirectory(f.path)
		if err != nil {
			select {
			case errCh <- err:
			case <-ctx.Done():
			}

			return
		}

		select {
		case <-ctx.Done():
			return
		case outputCh <- types.AgentSkillSourceItem(st):
		}
	}()

	return outputCh, errCh
}
