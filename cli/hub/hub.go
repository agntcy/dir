// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package hub

import (
	"context"

	"github.com/agntcy/dir/cli/hub/cmd"
	"github.com/agntcy/dir/cli/hub/cmd/options"
)

type ciscoHub struct{}

func NewCiscoHub() *ciscoHub { //nolint:revive
	return &ciscoHub{}
}

func (h *ciscoHub) Run(ctx context.Context, args []string) error {
	c := cmd.NewHubCommand(options.NewBaseOption())
	c.SetArgs(args)

	return c.ExecuteContext(ctx) //nolint: wrapcheck
}
