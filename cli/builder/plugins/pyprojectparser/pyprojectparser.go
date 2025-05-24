// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pyprojectparser

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/types"
)

type Project struct {
	Name        string   `toml:"name"`
	Version     string   `toml:"version"`
	Description string   `toml:"description"`
	Authors     []string `toml:"authors"`
}

type Metadata struct {
	Project Project `toml:"project"`
}

type PoetryMetadata struct {
	Tool struct {
		Poetry struct {
			Project
		} `toml:"poetry"`
	} `toml:"tool"`
}

type pyprojectparser struct {
	source string
}

func New(source string) types.Builder {
	return &pyprojectparser{
		source: source,
	}
}

func (p *pyprojectparser) Build(_ context.Context) (*coretypes.Agent, error) {
	// Search for pyproject.toml in the root directory and its subdirectories
	var pyprojectPath string

	err := filepath.Walk(p.source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Name() == "pyproject.toml" {
			pyprojectPath = path

			return filepath.SkipDir // Stop walking once the file is found
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking the path: %w", err)
	}

	// If the file was not found, return an error
	if pyprojectPath == "" {
		return nil, errors.New("pyproject.toml not found")
	}

	// Parse the pyproject.toml file
	var metadata Metadata
	if _, err := toml.DecodeFile(pyprojectPath, &metadata); err != nil {
		return nil, fmt.Errorf("error decoding pyproject.toml: %w", err)
	}

	// If empty, try to parse it as a Poetry project
	if metadata.Project.Name == "" && metadata.Project.Version == "" && metadata.Project.Description == "" {
		var poetryMetadata PoetryMetadata
		if _, err := toml.DecodeFile(pyprojectPath, &poetryMetadata); err != nil {
			return nil, fmt.Errorf("error decoding pyproject.toml: %w", err)
		}

		metadata = Metadata{
			Project: poetryMetadata.Tool.Poetry.Project,
		}
	}

	return &coretypes.Agent{
		Name:        metadata.Project.Name,
		Version:     metadata.Project.Version,
		Description: metadata.Project.Description,
		Authors:     metadata.Project.Authors,
	}, nil
}
