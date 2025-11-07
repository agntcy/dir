// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"strings"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show routing statistics and summary information",
	Long: `Show routing statistics and summary information for local records.

This command provides aggregated statistics about locally published records,
including record counts and label distribution.

Key Features:
- Record count: Total number of locally published records
- Label distribution: Frequency of each label across records
- Local-only: Shows statistics for local routing data only
- Fast: Uses local storage index for efficient counting

Usage examples:

1. Show local routing statistics:
   dirctl routing info

2. Output formats:
   # Get routing statistics as JSON
   dirctl routing info --output json
   
   # Get raw statistics data
   dirctl routing info --output raw

Note: For network-wide statistics, use 'dirctl routing search' with broad queries.
`,
	//nolint:gocritic // Lambda required due to signature mismatch - runInfoCommand doesn't use args
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runInfoCommand(cmd)
	},
}

func init() {
	// Add output format flags
	presenter.AddOutputFlags(infoCmd)
}

func runInfoCommand(cmd *cobra.Command) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Get output options
	outputOpts := presenter.GetOutputOptions(cmd)

	// Get all local records
	resultCh, err := c.List(cmd.Context(), &routingv1.ListRequest{
		// No queries = list all local records
	})
	if err != nil {
		return fmt.Errorf("failed to list local records: %w", err)
	}

	// Collect statistics
	stats := collectRoutingStatistics(resultCh)

	// Build structured result for all output formats
	result := map[string]interface{}{
		"totalRecords": stats.totalRecords,
		"skills":       stats.skillCounts,
		"locators":     stats.locatorCounts,
		"otherLabels":  stats.otherLabels,
	}

	// Use common PrintMessage for structured formats (json, jsonl, raw)
	if outputOpts.Format != presenter.FormatHuman {
		if err := presenter.PrintMessage(cmd, "routing statistics", "Routing statistics", result); err != nil {
			return fmt.Errorf("failed to print routing statistics: %w", err)
		}

		return nil
	}

	// Human-readable format
	presenter.Printf(cmd, "Local Routing Summary:\n\n")
	displayRoutingStatistics(cmd, stats)

	return nil
}

// routingStatistics holds collected routing statistics.
type routingStatistics struct {
	totalRecords  int
	skillCounts   map[string]int
	locatorCounts map[string]int
	otherLabels   map[string]int
}

// collectRoutingStatistics processes routing results and collects statistics.
func collectRoutingStatistics(resultCh <-chan *routingv1.ListResponse) *routingStatistics {
	stats := &routingStatistics{
		skillCounts:   make(map[string]int),
		locatorCounts: make(map[string]int),
		otherLabels:   make(map[string]int),
	}

	labelCounts := make(map[string]int)

	for result := range resultCh {
		stats.totalRecords++

		// Count and categorize labels
		for _, label := range result.GetLabels() {
			labelCounts[label]++
			categorizeLabel(label, stats)
		}
	}

	// Process other labels
	for label, count := range labelCounts {
		if !strings.HasPrefix(label, "/skills/") && !strings.HasPrefix(label, "/locators/") {
			stats.otherLabels[label] = count
		}
	}

	return stats
}

// categorizeLabel categorizes a label into the appropriate statistics bucket.
func categorizeLabel(label string, stats *routingStatistics) {
	switch {
	case strings.HasPrefix(label, "/skills/"):
		skillName := strings.TrimPrefix(label, "/skills/")
		stats.skillCounts[skillName]++
	case strings.HasPrefix(label, "/locators/"):
		locatorType := strings.TrimPrefix(label, "/locators/")
		stats.locatorCounts[locatorType]++
	}
}

// displayRoutingStatistics displays the collected statistics.
func displayRoutingStatistics(cmd *cobra.Command, stats *routingStatistics) {
	presenter.Printf(cmd, "📊 Record Statistics:\n")
	presenter.Printf(cmd, "  Total Records: %d\n\n", stats.totalRecords)

	if stats.totalRecords == 0 {
		displayEmptyStatistics(cmd)

		return
	}

	displaySkillStatistics(cmd, stats.skillCounts)
	displayLocatorStatistics(cmd, stats.locatorCounts)
	displayOtherLabels(cmd, stats.otherLabels)
	displayHelpfulTips(cmd)
}

// displayEmptyStatistics shows guidance when no records are found.
func displayEmptyStatistics(cmd *cobra.Command) {
	presenter.Printf(cmd, "No local records found.\n")
	presenter.Printf(cmd, "Use 'dirctl push' and 'dirctl routing publish' to add records.\n")
}

// displaySkillStatistics shows skill distribution.
func displaySkillStatistics(cmd *cobra.Command, skillCounts map[string]int) {
	if len(skillCounts) > 0 {
		presenter.Printf(cmd, "🎯 Skills Distribution:\n")

		for skill, count := range skillCounts {
			presenter.Printf(cmd, "  %s: %d record(s)\n", skill, count)
		}

		presenter.Printf(cmd, "\n")
	}
}

// displayLocatorStatistics shows locator distribution.
func displayLocatorStatistics(cmd *cobra.Command, locatorCounts map[string]int) {
	if len(locatorCounts) > 0 {
		presenter.Printf(cmd, "📍 Locators Distribution:\n")

		for locator, count := range locatorCounts {
			presenter.Printf(cmd, "  %s: %d record(s)\n", locator, count)
		}

		presenter.Printf(cmd, "\n")
	}
}

// displayOtherLabels shows other label types.
func displayOtherLabels(cmd *cobra.Command, otherLabels map[string]int) {
	if len(otherLabels) > 0 {
		presenter.Printf(cmd, "🏷️ Other Labels:\n")

		for label, count := range otherLabels {
			presenter.Printf(cmd, "  %s: %d record(s)\n", label, count)
		}

		presenter.Printf(cmd, "\n")
	}
}

// displayHelpfulTips shows usage suggestions.
func displayHelpfulTips(cmd *cobra.Command) {
	presenter.Printf(cmd, "💡 Tips:\n")
	presenter.Printf(cmd, "  - Use 'dirctl routing list --skill <skill>' to filter by skill\n")
	presenter.Printf(cmd, "  - Use 'dirctl routing search --skill <skill>' to find remote records\n")
}
