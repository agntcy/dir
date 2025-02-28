// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package crewai

// TODO: extract crewAI agent data from a file

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/agntcy/dir/cli/types"
	"github.com/ghetzel/go-stockutil/maputil"
	"gopkg.in/yaml.v3"
)

var keyTypeMapping = map[string]string{
	"role":            "agent",
	"goal":            "agent",
	"backstory":       "agent",
	"agent":           "task",
	"description":     "task",
	"expected_output": "task",
	"config":          "task",
	"human_input":     "task",
}

const (
	PluginName    = "crewai"
	PluginVersion = "v0.0.0"
)

type crewAI struct {
	path        string
	ignorePaths []string
}

func New(path string, ignore []string) types.Builder {
	return &crewAI{
		path:        path,
		ignorePaths: ignore,
	}
}

//nolint:gocognit,cyclop
func (c *crewAI) Build(_ context.Context) ([]*types.AgentExtension, error) {
	metadata := make(map[string]string)

	// open folder
	// parse agent data from filesystem
	err := filepath.WalkDir(c.path, func(fpath string, entry fs.DirEntry, _ error) error {
		// skip files in ignore list
		for _, ignorePath := range c.ignorePaths {
			if strings.Contains(fpath, ignorePath) {
				return nil
			}
		}
		// skip dirs and non-yaml crewAI files
		if entry == nil || !entry.Type().IsRegular() {
			return nil
		}

		switch filepath.Ext(fpath) {
		case ".yml", ".yaml":
			// only yaml files
		default:
			return nil
		}

		// open file
		fileData, err := os.ReadFile(fpath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}

		// load data from filem
		fileMeta := map[string]interface{}{}
		if err := yaml.Unmarshal(fileData, &fileMeta); err != nil {
			return fmt.Errorf("failed to parse file: %w", err)
		}

		// process loaded meta into final result and squash
		err = maputil.Walk(fileMeta, func(value interface{}, path []string, isLeaf bool) error {
			if !isLeaf || len(path) == 0 {
				return nil
			}

			// create non-prefixed key
			key := strings.Join(path, ".")

			// set key type prefix based on the attribute
			keyType, found := keyTypeMapping[path[len(path)-1]]
			if !found {
				return nil // skip non-crewai attributes
			}

			key = fmt.Sprintf("%s.%s", keyType, key)

			// validate if key already present to avoid same agent definition
			if _, exists := metadata[key]; exists {
				return fmt.Errorf("same agents defined multiple times: %s", key)
			}

			// write key
			metadata[key] = fmt.Sprintf("%v", value)

			// inject IO details
			if strings.Contains(key, "task.") {
				parts := strings.Split(key, ".")
				if len(parts) >= 2 { //nolint:mnd
					ioKey := "inputs." + parts[1]
					metadata[ioKey] = "string"
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to squash map: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process file tree: %w", err)
	}

	return []*types.AgentExtension{
		{
			Name:    PluginName,
			Version: PluginVersion,
			Data:    metadata,
		},
	}, nil
}
