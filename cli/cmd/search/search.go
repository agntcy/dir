// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"errors"
	"fmt"

	searchtypesv1alpha2 "github.com/agntcy/dir/api/search/v1alpha2"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "search",
	Short: "Search for records",
	Long: `Search for records in the directory using various filters and options.

Usage examples:

1. Search for records with specific filters and limit:

	dirctl search --limit 10 \
		--offset 0 \
		--query "name=my-agent-name" \
		--query "version=v1.0.0" \
		--query "skill-id=10201" \
		--query "skill-name=Text Completion" \
		--query "locator=docker-image:https://example.com/docker-image" \
		--query "extension=my-custom-extension-name:v1.0.0" 

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	ch, err := c.Search(cmd.Context(), &searchtypesv1alpha2.SearchRequest{
		Limit:   &opts.Limit,
		Offset:  &opts.Offset,
		Queries: opts.Query.ToAPIQueries(),
	})
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	for recordCid := range ch {
		if recordCid == "" {
			continue
		}

		presenter.Print(cmd, recordCid+"\n")
	}

	return nil
}
