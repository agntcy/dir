// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package base

import (
	"context"
	"time"

	"github.com/agntcy/dir/cli/types"
)

const (
	// ExtensionName is the name of the extension
	ExtensionName = "base"
	// ExtensionVersion is the version of the extension
	ExtensionVersion = "v0.0.0"
)

type ExtensionSpecs struct {
	CreatedAt string   `json:"created_at,omitempty"`
	Authors   []string `json:"authors,omitempty"`
}

type base struct {
	authors []string
}

func New(authors []string) types.ExtensionBuilder {
	return &base{
		authors: authors,
	}
}

func (b *base) Build(_ context.Context) (*types.AgentExtension, error) {
	return &types.AgentExtension{
		Name:    ExtensionName,
		Version: ExtensionVersion,
		Specs: ExtensionSpecs{
			CreatedAt: time.Now().Format(time.RFC3339),
			Authors:   b.authors,
		},
	}, nil
}
