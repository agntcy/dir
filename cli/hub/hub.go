package hub

import (
	"context"

	"github.com/agntcy/dir/cli/hub/cmd"
	"github.com/agntcy/dir/cli/hub/cmd/options"
)

type ciscoHub struct{}

func NewCiscoHub() *ciscoHub {
	return &ciscoHub{}
}

func (h *ciscoHub) Run(ctx context.Context, args []string) error {
	c := cmd.NewHubCommand(options.NewBaseOption())
	c.SetArgs(args)
	return c.ExecuteContext(ctx)
}
