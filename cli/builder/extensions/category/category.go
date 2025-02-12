// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package category

import (
	"context"

	"github.com/agntcy/dir/cli/types"
)

const (
	// ExtensionName is the name of the extension
	ExtensionName = "category"
	// ExtensionVersion is the version of the extension
	ExtensionVersion = "v0.0.0"
)

type ExtensionSpecs struct {
	Categories []string `json:"categories"`
}

type category struct {
	categories []string
}

func New(categories []string) types.ExtensionBuilder {
	return &category{
		categories: categories,
	}
}

func (c *category) Build(_ context.Context) (*types.AgentExtension, error) {
	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: ExtensionSpecs{
			Categories: c.categories,
		},
	}, nil
}
