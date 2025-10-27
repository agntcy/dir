// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"bytes"
	"strings"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCategorizeLabel tests label categorization logic.
func TestCategorizeLabel(t *testing.T) {
	tests := []struct {
		name           string
		label          string
		expectedSkills map[string]int
		expectedLocs   map[string]int
	}{
		{
			name:           "skill label",
			label:          "/skills/AI",
			expectedSkills: map[string]int{"AI": 1},
			expectedLocs:   map[string]int{},
		},
		{
			name:           "skill with spaces",
			label:          "/skills/Natural Language Processing",
			expectedSkills: map[string]int{"Natural Language Processing": 1},
			expectedLocs:   map[string]int{},
		},
		{
			name:           "locator label",
			label:          "/locators/docker-image",
			expectedSkills: map[string]int{},
			expectedLocs:   map[string]int{"docker-image": 1},
		},
		{
			name:           "locator with path",
			label:          "/locators/http/endpoint",
			expectedSkills: map[string]int{},
			expectedLocs:   map[string]int{"http/endpoint": 1},
		},
		{
			name:           "other label - ignored",
			label:          "/domains/healthcare",
			expectedSkills: map[string]int{},
			expectedLocs:   map[string]int{},
		},
		{
			name:           "plain label - ignored",
			label:          "custom-label",
			expectedSkills: map[string]int{},
			expectedLocs:   map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &routingStatistics{
				skillCounts:   make(map[string]int),
				locatorCounts: make(map[string]int),
				otherLabels:   make(map[string]int),
			}

			categorizeLabel(tt.label, stats)

			assert.Equal(t, tt.expectedSkills, stats.skillCounts)
			assert.Equal(t, tt.expectedLocs, stats.locatorCounts)
		})
	}
}

// TestCategorizeLabel_MultipleCallsSameLabel tests counting logic.
func TestCategorizeLabel_MultipleCallsSameLabel(t *testing.T) {
	stats := &routingStatistics{
		skillCounts:   make(map[string]int),
		locatorCounts: make(map[string]int),
		otherLabels:   make(map[string]int),
	}

	// Call three times with same skill
	categorizeLabel("/skills/AI", stats)
	categorizeLabel("/skills/AI", stats)
	categorizeLabel("/skills/AI", stats)

	assert.Equal(t, map[string]int{"AI": 3}, stats.skillCounts)
	assert.Equal(t, map[string]int{}, stats.locatorCounts)
}

// TestCategorizeLabel_MultipleDifferentLabels tests mixed labels.
func TestCategorizeLabel_MultipleDifferentLabels(t *testing.T) {
	stats := &routingStatistics{
		skillCounts:   make(map[string]int),
		locatorCounts: make(map[string]int),
		otherLabels:   make(map[string]int),
	}

	categorizeLabel("/skills/AI", stats)
	categorizeLabel("/skills/ML", stats)
	categorizeLabel("/locators/docker-image", stats)
	categorizeLabel("/locators/http", stats)

	assert.Equal(t, map[string]int{"AI": 1, "ML": 1}, stats.skillCounts)
	assert.Equal(t, map[string]int{"docker-image": 1, "http": 1}, stats.locatorCounts)
}

// TestCollectRoutingStatistics_EmptyChannel tests with no records.
func TestCollectRoutingStatistics_EmptyChannel(t *testing.T) {
	ch := make(chan *routingv1.ListResponse)
	close(ch)

	stats := collectRoutingStatistics(ch)

	assert.Equal(t, 0, stats.totalRecords)
	assert.Empty(t, stats.skillCounts)
	assert.Empty(t, stats.locatorCounts)
	assert.Empty(t, stats.otherLabels)
}

// TestCollectRoutingStatistics_SingleRecord tests with one record.
func TestCollectRoutingStatistics_SingleRecord(t *testing.T) {
	ch := make(chan *routingv1.ListResponse, 1)
	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "test-cid"},
		Labels:    []string{"/skills/AI", "/locators/docker-image"},
	}

	close(ch)

	stats := collectRoutingStatistics(ch)

	assert.Equal(t, 1, stats.totalRecords)
	assert.Equal(t, map[string]int{"AI": 1}, stats.skillCounts)
	assert.Equal(t, map[string]int{"docker-image": 1}, stats.locatorCounts)
	assert.Empty(t, stats.otherLabels)
}

// TestCollectRoutingStatistics_MultipleRecords tests with multiple records.
func TestCollectRoutingStatistics_MultipleRecords(t *testing.T) {
	ch := make(chan *routingv1.ListResponse, 3)
	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "cid1"},
		Labels:    []string{"/skills/AI", "/locators/docker-image"},
	}

	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "cid2"},
		Labels:    []string{"/skills/AI", "/skills/ML"},
	}

	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "cid3"},
		Labels:    []string{"/skills/ML", "/locators/http"},
	}

	close(ch)

	stats := collectRoutingStatistics(ch)

	assert.Equal(t, 3, stats.totalRecords)
	assert.Equal(t, map[string]int{"AI": 2, "ML": 2}, stats.skillCounts)
	assert.Equal(t, map[string]int{"docker-image": 1, "http": 1}, stats.locatorCounts)
	assert.Empty(t, stats.otherLabels)
}

// TestCollectRoutingStatistics_WithOtherLabels tests other label categories.
func TestCollectRoutingStatistics_WithOtherLabels(t *testing.T) {
	ch := make(chan *routingv1.ListResponse, 2)
	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "cid1"},
		Labels:    []string{"/skills/AI", "/domains/healthcare", "/custom/label"},
	}

	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "cid2"},
		Labels:    []string{"/domains/healthcare", "/modules/runtime"},
	}

	close(ch)

	stats := collectRoutingStatistics(ch)

	assert.Equal(t, 2, stats.totalRecords)
	assert.Equal(t, map[string]int{"AI": 1}, stats.skillCounts)
	assert.Empty(t, stats.locatorCounts)
	assert.Equal(t, map[string]int{
		"/domains/healthcare": 2,
		"/custom/label":       1,
		"/modules/runtime":    1,
	}, stats.otherLabels)
}

// TestCollectRoutingStatistics_RecordWithNoLabels tests records without labels.
func TestCollectRoutingStatistics_RecordWithNoLabels(t *testing.T) {
	ch := make(chan *routingv1.ListResponse, 2)
	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "cid1"},
		Labels:    []string{},
	}

	ch <- &routingv1.ListResponse{
		RecordRef: &corev1.RecordRef{Cid: "cid2"},
		Labels:    nil,
	}

	close(ch)

	stats := collectRoutingStatistics(ch)

	assert.Equal(t, 2, stats.totalRecords)
	assert.Empty(t, stats.skillCounts)
	assert.Empty(t, stats.locatorCounts)
	assert.Empty(t, stats.otherLabels)
}

// TestDisplayEmptyStatistics tests empty statistics display.
func TestDisplayEmptyStatistics(t *testing.T) {
	cmd := &cobra.Command{}

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	displayEmptyStatistics(cmd)

	output := stdout.String()
	assert.Contains(t, output, "No local records found")
	assert.Contains(t, output, "dirctl push")
	assert.Contains(t, output, "dirctl routing publish")
}

// TestDisplaySkillStatistics tests skill statistics display.
//
//nolint:dupl // Similar test structure for different display functions is intentional for clarity
func TestDisplaySkillStatistics(t *testing.T) {
	tests := []struct {
		name     string
		skills   map[string]int
		expected []string
	}{
		{
			name:     "empty skills",
			skills:   map[string]int{},
			expected: []string{},
		},
		{
			name:     "single skill",
			skills:   map[string]int{"AI": 5},
			expected: []string{"Skills Distribution", "AI: 5 record(s)"},
		},
		{
			name:     "multiple skills",
			skills:   map[string]int{"AI": 3, "ML": 2},
			expected: []string{"Skills Distribution", "AI: 3 record(s)", "ML: 2 record(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			displaySkillStatistics(cmd, tt.skills)

			output := stdout.String()
			for _, exp := range tt.expected {
				assert.Contains(t, output, exp)
			}

			if len(tt.skills) == 0 {
				assert.Empty(t, output)
			}
		})
	}
}

// TestDisplayLocatorStatistics tests locator statistics display.
//
//nolint:dupl // Similar test structure for different display functions is intentional for clarity
func TestDisplayLocatorStatistics(t *testing.T) {
	tests := []struct {
		name     string
		locators map[string]int
		expected []string
	}{
		{
			name:     "empty locators",
			locators: map[string]int{},
			expected: []string{},
		},
		{
			name:     "single locator",
			locators: map[string]int{"docker-image": 3},
			expected: []string{"Locators Distribution", "docker-image: 3 record(s)"},
		},
		{
			name:     "multiple locators",
			locators: map[string]int{"docker-image": 2, "http": 1},
			expected: []string{"Locators Distribution", "docker-image: 2 record(s)", "http: 1 record(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			displayLocatorStatistics(cmd, tt.locators)

			output := stdout.String()
			for _, exp := range tt.expected {
				assert.Contains(t, output, exp)
			}

			if len(tt.locators) == 0 {
				assert.Empty(t, output)
			}
		})
	}
}

// TestDisplayOtherLabels tests other labels display.
//
//nolint:dupl // Similar test structure for different display functions is intentional for clarity
func TestDisplayOtherLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]int
		expected []string
	}{
		{
			name:     "empty labels",
			labels:   map[string]int{},
			expected: []string{},
		},
		{
			name:     "single label",
			labels:   map[string]int{"/domains/healthcare": 2},
			expected: []string{"Other Labels", "/domains/healthcare: 2 record(s)"},
		},
		{
			name:     "multiple labels",
			labels:   map[string]int{"/domains/healthcare": 2, "/modules/runtime": 1},
			expected: []string{"Other Labels", "/domains/healthcare: 2 record(s)", "/modules/runtime: 1 record(s)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)

			displayOtherLabels(cmd, tt.labels)

			output := stdout.String()
			for _, exp := range tt.expected {
				assert.Contains(t, output, exp)
			}

			if len(tt.labels) == 0 {
				assert.Empty(t, output)
			}
		})
	}
}

// TestDisplayHelpfulTips tests helpful tips display.
func TestDisplayHelpfulTips(t *testing.T) {
	cmd := &cobra.Command{}

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	displayHelpfulTips(cmd)

	output := stdout.String()
	assert.Contains(t, output, "Tips")
	assert.Contains(t, output, "dirctl routing list --skill")
	assert.Contains(t, output, "dirctl routing search --skill")
}

// TestDisplayRoutingStatistics_EmptyStats tests display with zero records.
func TestDisplayRoutingStatistics_EmptyStats(t *testing.T) {
	cmd := &cobra.Command{}

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	stats := &routingStatistics{
		totalRecords:  0,
		skillCounts:   make(map[string]int),
		locatorCounts: make(map[string]int),
		otherLabels:   make(map[string]int),
	}

	displayRoutingStatistics(cmd, stats)

	output := stdout.String()
	assert.Contains(t, output, "Total Records: 0")
	assert.Contains(t, output, "No local records found")
	// Should not display other sections
	assert.NotContains(t, output, "Skills Distribution")
	assert.NotContains(t, output, "Locators Distribution")
	assert.NotContains(t, output, "Other Labels")
	assert.NotContains(t, output, "Tips")
}

// TestDisplayRoutingStatistics_WithData tests display with actual statistics.
func TestDisplayRoutingStatistics_WithData(t *testing.T) {
	cmd := &cobra.Command{}

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	stats := &routingStatistics{
		totalRecords: 5,
		skillCounts: map[string]int{
			"AI": 3,
			"ML": 2,
		},
		locatorCounts: map[string]int{
			"docker-image": 2,
		},
		otherLabels: map[string]int{
			"/domains/healthcare": 1,
		},
	}

	displayRoutingStatistics(cmd, stats)

	output := stdout.String()
	// Check all sections are present
	assert.Contains(t, output, "Total Records: 5")
	assert.Contains(t, output, "Skills Distribution")
	assert.Contains(t, output, "AI: 3 record(s)")
	assert.Contains(t, output, "ML: 2 record(s)")
	assert.Contains(t, output, "Locators Distribution")
	assert.Contains(t, output, "docker-image: 2 record(s)")
	assert.Contains(t, output, "Other Labels")
	assert.Contains(t, output, "/domains/healthcare: 1 record(s)")
	assert.Contains(t, output, "Tips")
}

// TestDisplayRoutingStatistics_OnlySkills tests display with only skills.
func TestDisplayRoutingStatistics_OnlySkills(t *testing.T) {
	cmd := &cobra.Command{}

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	stats := &routingStatistics{
		totalRecords: 2,
		skillCounts: map[string]int{
			"AI": 2,
		},
		locatorCounts: make(map[string]int),
		otherLabels:   make(map[string]int),
	}

	displayRoutingStatistics(cmd, stats)

	output := stdout.String()
	assert.Contains(t, output, "Total Records: 2")
	assert.Contains(t, output, "Skills Distribution")
	assert.Contains(t, output, "AI: 2 record(s)")
	assert.NotContains(t, output, "Locators Distribution")
	assert.NotContains(t, output, "Other Labels")
	assert.Contains(t, output, "Tips")
}

// TestInfoCmd_Initialization tests that infoCmd is properly initialized.
func TestInfoCmd_Initialization(t *testing.T) {
	assert.NotNil(t, infoCmd)
	assert.Equal(t, "info", infoCmd.Use)
	assert.NotEmpty(t, infoCmd.Short)
	assert.NotEmpty(t, infoCmd.Long)
	assert.NotNil(t, infoCmd.RunE)

	// Check that examples are in the Long description
	assert.Contains(t, infoCmd.Long, "dirctl routing info")
}

// TestRoutingStatistics_Structure tests the statistics structure.
func TestRoutingStatistics_Structure(t *testing.T) {
	stats := &routingStatistics{
		totalRecords:  10,
		skillCounts:   map[string]int{"AI": 5},
		locatorCounts: map[string]int{"docker": 3},
		otherLabels:   map[string]int{"/custom": 2},
	}

	assert.Equal(t, 10, stats.totalRecords)
	assert.Equal(t, 5, stats.skillCounts["AI"])
	assert.Equal(t, 3, stats.locatorCounts["docker"])
	assert.Equal(t, 2, stats.otherLabels["/custom"])
}

// TestCollectRoutingStatistics_LargeDataset tests with many records.
func TestCollectRoutingStatistics_LargeDataset(t *testing.T) {
	ch := make(chan *routingv1.ListResponse, 100)

	// Add 100 records with varying labels
	for i := range 100 {
		labels := []string{"/skills/AI"}
		if i%2 == 0 {
			labels = append(labels, "/locators/docker-image")
		}

		if i%3 == 0 {
			labels = append(labels, "/domains/healthcare")
		}

		ch <- &routingv1.ListResponse{
			RecordRef: &corev1.RecordRef{Cid: "cid-" + string(rune(i))},
			Labels:    labels,
		}
	}

	close(ch)

	stats := collectRoutingStatistics(ch)

	assert.Equal(t, 100, stats.totalRecords)
	assert.Equal(t, 100, stats.skillCounts["AI"])
	assert.Equal(t, 50, stats.locatorCounts["docker-image"])
	assert.Equal(t, 34, stats.otherLabels["/domains/healthcare"]) // 100/3 rounded up
}

// TestCategorizeLabel_EdgeCases tests edge cases in label categorization.
func TestCategorizeLabel_EdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		label string
	}{
		{"empty string", ""},
		{"just slash", "/"},
		{"skills prefix only", "/skills/"},
		{"locators prefix only", "/locators/"},
		{"multiple slashes", "/skills//AI//ML"},
		{"trailing slash", "/skills/AI/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &routingStatistics{
				skillCounts:   make(map[string]int),
				locatorCounts: make(map[string]int),
				otherLabels:   make(map[string]int),
			}

			// Should not panic
			require.NotPanics(t, func() {
				categorizeLabel(tt.label, stats)
			})
		})
	}
}

// TestDisplayFunctions_WithEmojis tests that display functions include emojis.
func TestDisplayFunctions_WithEmojis(t *testing.T) {
	cmd := &cobra.Command{}

	var stdout bytes.Buffer
	cmd.SetOut(&stdout)

	stats := &routingStatistics{
		totalRecords:  1,
		skillCounts:   map[string]int{"AI": 1},
		locatorCounts: map[string]int{"docker": 1},
		otherLabels:   map[string]int{"/custom": 1},
	}

	displayRoutingStatistics(cmd, stats)

	output := stdout.String()
	// Check for emojis (they make the output more user-friendly)
	assert.True(t, strings.Contains(output, "📊") || strings.Contains(output, "Record Statistics"))
	assert.True(t, strings.Contains(output, "🎯") || strings.Contains(output, "Skills"))
	assert.True(t, strings.Contains(output, "📍") || strings.Contains(output, "Locators"))
	assert.True(t, strings.Contains(output, "🏷️") || strings.Contains(output, "Other Labels") || strings.Contains(output, "Labels"))
	assert.True(t, strings.Contains(output, "💡") || strings.Contains(output, "Tips"))
}
