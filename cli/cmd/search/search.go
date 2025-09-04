// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package search

import (
	"errors"
	"fmt"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "search",
	Short: "Search for records",
	Long: `Search for records in the directory using various filters and options.

Usage examples:

1. Basic search with specific filters and limit:

	dirctl search --limit 10 \
		--offset 0 \
		--query "name=my-agent-name" \
		--query "version=v1.0.0" \
		--query "skill-id=10201" \
		--query "skill-name=Text Completion" \
		--query "locator=docker-image:https://example.com/docker-image" \
		--query "extension=my-custom-extension-name:v1.0.0" 

2. Wildcard search examples:

	# Find all web-related agents
	dirctl search --query "name=web*"
	
	# Find all v1.x versions
	dirctl search --query "version=v1.*"
	
	# Find agents with Python or JavaScript skills
	dirctl search --query "skill-name=python*" --query "skill-name=*script"
	
	# Find agents with HTTP-based locators
	dirctl search --query "locator=http*"
	
	# Find agents with plugin extensions
	dirctl search --query "extension=*-plugin*"

3. Question mark wildcard (? matches exactly one character):

	# Find version v1.0.x where x is any single digit
	dirctl search --query "version=v1.0.?"
	
	# Find agents with 3-character names ending in "api"
	dirctl search --query "name=???api"
	
	# Find skills with single character variations
	dirctl search --query "skill-name=Pytho?"

4. List wildcards ([] matches any character within brackets):

	# Find agents with numeric suffixes
	dirctl search --query "name=agent-[0-9]"
	
	# Find versions starting with v followed by any digit
	dirctl search --query "version=v[0-9].*"
	
	# Find skills starting with uppercase letters A-M
	dirctl search --query "skill-name=[A-M]*"
	
	# Find locators with specific protocols
	dirctl search --query "locator=[hf]tt[ps]*"

5. Complex wildcard patterns:

	# Find API services with v2 versions
	dirctl search --query "name=api-*-service" --query "version=v2.*"
	
	# Find machine learning agents
	dirctl search --query "skill-name=*machine*learning*"
	
	# Find agents with container locators
	dirctl search --query "locator=*docker*" --query "locator=*container*"
	
	# Combine different wildcard types
	dirctl search --query "name=web-[0-9]?" --query "version=v?.*.?"

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

	ch, err := c.Search(cmd.Context(), &searchv1.SearchRequest{
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
