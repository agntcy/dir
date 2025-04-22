// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package hub

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/agntcy/dir/hub/cmd"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/config"
	"github.com/spf13/cobra"
)

type ciscoHub struct{}

func NewCiscoHub() *ciscoHub { //nolint:revive
	return &ciscoHub{}
}

func (h *ciscoHub) Run(ctx context.Context, args []string) error {
	cobra.EnableTraverseRunHooks = true

	err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	opts := options.NewBaseOption()
	c := cmd.NewHubCommand(opts)

	if err = opts.Register(); err != nil {
		return fmt.Errorf("failed to register hub options: %w", err)
	}

	errBuf := bytes.NewBuffer([]byte{})
	outBuf := bytes.NewBuffer([]byte{})

	c.SetErr(errBuf)
	c.SetOut(outBuf)
	c.SetArgs(args)

	var outStr string

	if err = c.ExecuteContext(ctx); err != nil {
		r := regexp.MustCompile("(\n\\s*)hub")
		outStr = r.ReplaceAllString(outStr, "\n dirctl hub")

		fmt.Fprintln(os.Stderr, errBuf.String())
	} else {
		outStr = outBuf.String()
	}

	fmt.Fprintln(os.Stdout, outStr)

	return nil
}
