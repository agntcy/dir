// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"errors"
	"fmt"
	"io"
	"strings"

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
		--query "agent-name:my-agent-name" \
		--query "agent-version:v1.0.0" \
		--query "skill-id:10201" \
		--query "skill-name:Text Completion" \
		--query "locator:docker-image:https://example.com/docker-image" \
		--query "extension:my-custom-extension-name:v1.0.0" 

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	if opts.Limit <= 0 {
		return errors.New("limit must be greater than 0")
	}

	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	queries := make([]*searchtypesv1alpha2.RecordQuery, 0)

	for _, q := range opts.Query {
		if q == "" {
			return errors.New("query cannot be empty")
		}

		query := strings.SplitN(q, ":", 2) //nolint:mnd

		if len(query) != 2 { //nolint:mnd
			return fmt.Errorf("invalid query format: %s, expected format is 'type:value'", q)
		}

		if _, ok := searchtypesv1alpha2.RecordQueryType_value[query[0]]; !ok {
			return fmt.Errorf("invalid query type: %s, valid types are: %v", query[0], searchtypesv1alpha2.ValidQueryTypes)
		}

		if query[1] == "" {
			return fmt.Errorf("query value cannot be empty for type: %s", query[0])
		}

		queries = append(queries, &searchtypesv1alpha2.RecordQuery{
			Type:  searchtypesv1alpha2.RecordQueryType(searchtypesv1alpha2.RecordQueryType_value[query[0]]),
			Value: query[1],
		})
	}

	reader, err := c.SearchV1alpha2(cmd.Context(), &searchtypesv1alpha2.SearchRequest{
		Limit:   &opts.Limit,
		Offset:  &opts.Offset,
		Queries: queries,
	})
	if err != nil {
		return fmt.Errorf("failed to searchh: %w", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read search response: %w", err)
	}

	presenter.Print(cmd, string(data))

	return nil
}
