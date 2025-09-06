// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLabels(t *testing.T) {
	testCases := []struct {
		name           string
		record         *corev1.Record
		expectedLabels []string
	}{
		{
			name: "v3_record_with_all_types",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "test-agent",
						Version:     "1.0.0",
						Description: "Test agent with all types",
						Skills: []*objectsv3.Skill{
							{Name: "AI"},
							{Name: "AI/ML"},
							{Name: "web-development"},
						},
						Extensions: []*objectsv3.Extension{
							{Name: "schema.oasf.agntcy.org/domains/technology"},
							{Name: "schema.oasf.agntcy.org/domains/healthcare"},
							{Name: "schema.oasf.agntcy.org/features/text-processing"},
							{Name: "schema.oasf.agntcy.org/features/image-generation"},
							{Name: "other.metadata"}, // Should be ignored
						},
						Locators: []*objectsv3.Locator{
							{Type: "docker-image"},
							{Type: "git-repo"},
						},
					},
				},
			},
			expectedLabels: []string{
				"/skills/AI",
				"/skills/AI/ML",
				"/skills/web-development",
				"/domains/technology",
				"/domains/healthcare",
				"/features/text-processing",
				"/features/image-generation",
				"/locators/docker-image",
				"/locators/git-repo",
			},
		},
		{
			name: "v3_record_skills_only",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "blockchain-agent",
						Version:     "1.0.0",
						Description: "Blockchain specialist agent",
						Skills: []*objectsv3.Skill{
							{Name: "blockchain"},
							{Name: "smart-contracts"},
						},
						Extensions: nil,
						Locators:   nil,
					},
				},
			},
			expectedLabels: []string{
				"/skills/blockchain",
				"/skills/smart-contracts",
			},
		},
		{
			name: "v3_record_extensions_only",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "finance-agent",
						Version:     "1.0.0",
						Description: "Finance domain agent",
						Skills:      nil,
						Extensions: []*objectsv3.Extension{
							{Name: "schema.oasf.agntcy.org/domains/finance"},
							{Name: "schema.oasf.agntcy.org/features/api-integration"},
						},
						Locators: nil,
					},
				},
			},
			expectedLabels: []string{
				"/domains/finance",
				"/features/api-integration",
			},
		},
		{
			name: "v3_record_locators_only",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "deployment-agent",
						Version:     "1.0.0",
						Description: "Deployment specialist agent",
						Skills:      nil,
						Extensions:  nil,
						Locators: []*objectsv3.Locator{
							{Type: "docker-image"},
							{Type: "helm-chart"},
						},
					},
				},
			},
			expectedLabels: []string{
				"/locators/docker-image",
				"/locators/helm-chart",
			},
		},
		{
			name: "v3_record_empty",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "empty-agent",
						Version:     "1.0.0",
						Description: "Agent with no skills, extensions, or locators",
						Skills:      nil,
						Extensions:  nil,
						Locators:    nil,
					},
				},
			},
			expectedLabels: []string{},
		},
		{
			name: "v3_record_with_ignored_extensions",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "research-agent",
						Version:     "1.0.0",
						Description: "Research agent with mixed extensions",
						Skills: []*objectsv3.Skill{
							{Name: "data-science"},
						},
						Extensions: []*objectsv3.Extension{
							{Name: "metadata.version"},                        // Ignored - not domain.* or features.*
							{Name: "config.settings"},                         // Ignored
							{Name: "schema.oasf.agntcy.org/domains/research"}, // Included
						},
						Locators: nil,
					},
				},
			},
			expectedLabels: []string{
				"/skills/data-science",
				"/domains/research",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			labels := GetLabels(tc.record)

			// Sort both slices for comparison
			assert.ElementsMatch(t, tc.expectedLabels, labels)
		})
	}
}

func TestGetLabels_EdgeCases(t *testing.T) {
	t.Run("nil_record", func(t *testing.T) {
		labels := GetLabels(nil)
		assert.Nil(t, labels)
	})

	t.Run("record_with_nil_v3", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V3{V3: nil},
		}
		labels := GetLabels(record)
		assert.Nil(t, labels)
	})

	t.Run("unsupported_record_version", func(t *testing.T) {
		record := &corev1.Record{
			Data: nil, // No V3 field
		}
		labels := GetLabels(record)
		assert.Nil(t, labels)
	})

	t.Run("empty_skill_names", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V3{
				V3: &objectsv3.Record{
					Name:        "test-agent",
					Version:     "1.0.0",
					Description: "Agent with empty skill names",
					Skills: []*objectsv3.Skill{
						{Name: ""}, // Empty name
						{Name: "valid-skill"},
						{Name: ""}, // Another empty name
					},
				},
			},
		}
		labels := GetLabels(record)

		// Should include empty skill names (they get prefixed)
		expectedLabels := []string{
			"/skills/",
			"/skills/valid-skill",
			"/skills/",
		}
		assert.ElementsMatch(t, expectedLabels, labels)
	})

	t.Run("empty_locator_types", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V3{
				V3: &objectsv3.Record{
					Name:        "test-agent",
					Version:     "1.0.0",
					Description: "Agent with empty locator types",
					Locators: []*objectsv3.Locator{
						{Type: ""},
						{Type: "docker-image"},
					},
				},
			},
		}
		labels := GetLabels(record)

		expectedLabels := []string{
			"/locators/",
			"/locators/docker-image",
		}
		assert.ElementsMatch(t, expectedLabels, labels)
	})

	t.Run("malformed_extension_names", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V3{
				V3: &objectsv3.Record{
					Name:        "test-agent",
					Version:     "1.0.0",
					Description: "Agent with malformed extension names",
					Extensions: []*objectsv3.Extension{
						{Name: "schema.oasf.agntcy.org/domains/"}, // Ends with dot
						{Name: "domain"}, // Missing dot - ignored
						{Name: "schema.oasf.agntcy.org/features/"},      // Ends with dot
						{Name: "schema.oasf.agntcy.org/domains/valid"},  // Valid
						{Name: "schema.oasf.agntcy.org/features/valid"}, // Valid
					},
				},
			},
		}
		labels := GetLabels(record)

		expectedLabels := []string{
			"/domains/",       // schema.oasf.agntcy.org/domains/ becomes /domains/
			"/features/",      // schema.oasf.agntcy.org/features/ becomes /features/
			"/domains/valid",  // schema.oasf.agntcy.org/domains/valid becomes /domains/valid
			"/features/valid", // schema.oasf.agntcy.org/features/valid becomes /features/valid
		}
		assert.ElementsMatch(t, expectedLabels, labels)
	})
}

func TestGetLabels_RealWorldScenarios(t *testing.T) {
	t.Run("ai_ml_agent", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V3{
				V3: &objectsv3.Record{
					Name:        "ai-ml-agent",
					Version:     "1.0.0",
					Description: "Advanced AI/ML agent",
					Skills: []*objectsv3.Skill{
						{Name: "AI"},
						{Name: "AI/ML"},
						{Name: "AI/NLP"},
						{Name: "python"},
						{Name: "tensorflow"},
					},
					Extensions: []*objectsv3.Extension{
						{Name: "schema.oasf.agntcy.org/domains/technology"},
						{Name: "schema.oasf.agntcy.org/domains/research"},
						{Name: "schema.oasf.agntcy.org/features/text-processing"},
						{Name: "schema.oasf.agntcy.org/features/model-training"},
					},
					Locators: []*objectsv3.Locator{
						{Type: "docker-image"},
						{Type: "git-repo"},
					},
				},
			},
		}

		labels := GetLabels(record)

		expectedLabels := []string{
			"/skills/AI",
			"/skills/AI/ML",
			"/skills/AI/NLP",
			"/skills/python",
			"/skills/tensorflow",
			"/domains/technology",
			"/domains/research",
			"/features/text-processing",
			"/features/model-training",
			"/locators/docker-image",
			"/locators/git-repo",
		}

		assert.ElementsMatch(t, expectedLabels, labels)
		assert.Len(t, labels, len(expectedLabels))
	})

	t.Run("web_development_agent", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V3{
				V3: &objectsv3.Record{
					Name:        "web-dev-agent",
					Version:     "1.0.0",
					Description: "Web development specialist agent",
					Skills: []*objectsv3.Skill{
						{Name: "web-development"},
						{Name: "javascript"},
						{Name: "react"},
						{Name: "nodejs"},
					},
					Extensions: []*objectsv3.Extension{
						{Name: "schema.oasf.agntcy.org/domains/technology"},
						{Name: "schema.oasf.agntcy.org/features/frontend"},
						{Name: "schema.oasf.agntcy.org/features/backend"},
					},
					Locators: []*objectsv3.Locator{
						{Type: "docker-image"},
						{Type: "npm-package"},
					},
				},
			},
		}

		labels := GetLabels(record)

		expectedLabels := []string{
			"/skills/web-development",
			"/skills/javascript",
			"/skills/react",
			"/skills/nodejs",
			"/domains/technology",
			"/features/frontend",
			"/features/backend",
			"/locators/docker-image",
			"/locators/npm-package",
		}

		assert.ElementsMatch(t, expectedLabels, labels)
	})
}

// Test that GetLabels works with the adapter pattern correctly.
func TestGetLabels_AdapterIntegration(t *testing.T) {
	// This test ensures that GetLabels correctly uses the adapter pattern
	// and can handle different record versions if they exist
	t.Run("v3_record_through_adapter", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V3{
				V3: &objectsv3.Record{
					Name:        "adapter-test-agent",
					Version:     "1.0.0",
					Description: "Agent for testing adapter pattern",
					Skills: []*objectsv3.Skill{
						{Name: "adapter-test"},
					},
				},
			},
		}

		labels := GetLabels(record)
		require.NotNil(t, labels)
		assert.Contains(t, labels, "/skills/adapter-test")
	})
}
