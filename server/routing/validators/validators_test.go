// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillValidator_Validate(t *testing.T) {
	validator := &SkillValidator{}

	tests := []struct {
		name      string
		key       string
		value     []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid skills key with category and class",
			key:       "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid skills key with value",
			key:       "/skills/ai/machine-learning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2",
			value:     []byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
			wantError: false,
		},
		{
			name:      "invalid namespace",
			key:       "/domains/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid namespace: expected skills, got domains",
		},
		{
			name:      "missing skill path",
			key:       "/skills/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
		{
			name:      "valid single skill path",
			key:       "/skills/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "empty skill path component",
			key:       "/skills//golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "skill path component cannot be empty at position 1",
		},
		{
			name:      "empty skill path component in middle",
			key:       "/skills/programming//advanced/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "skill path component cannot be empty at position 2",
		},
		{
			name:      "invalid CID format",
			key:       "/skills/programming/golang/invalid-cid/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid CID format",
		},
		{
			name:      "invalid value CID",
			key:       "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte("invalid-cid-value"),
			wantError: true,
			errorMsg:  "invalid CID in value",
		},
		{
			name:      "missing CID",
			key:       "/skills/programming/golang//Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "missing CID in key",
		},
		{
			name:      "missing PeerID",
			key:       "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.key, tt.value)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:dupl // Similar test structure is intentional for different validators
func TestDomainValidator_Validate(t *testing.T) {
	validator := &DomainValidator{}

	tests := []struct {
		name      string
		key       string
		value     []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid domains key with single domain",
			key:       "/domains/ai/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid domains key with nested domain path",
			key:       "/domains/ai/machine-learning/nlp/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid domains key with value",
			key:       "/domains/software/web-development/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
			value:     []byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
			wantError: false,
		},
		{
			name:      "invalid namespace",
			key:       "/skills/ai/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid namespace: expected domains, got skills",
		},
		{
			name:      "missing domain path",
			key:       "/domains/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
		{
			name:      "invalid CID format",
			key:       "/domains/ai/invalid-cid/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid CID format",
		},
		{
			name:      "invalid value CID",
			key:       "/domains/ai/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte("invalid-cid-value"),
			wantError: true,
			errorMsg:  "invalid CID in value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.key, tt.value)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:dupl // Similar test structure is intentional for different validators
func TestFeatureValidator_Validate(t *testing.T) {
	validator := &FeatureValidator{}

	tests := []struct {
		name      string
		key       string
		value     []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid features key with single feature",
			key:       "/features/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid features key with nested feature path",
			key:       "/features/ai/reasoning/logical/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid features key with value",
			key:       "/features/search/semantic/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
			value:     []byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
			wantError: false,
		},
		{
			name:      "invalid namespace",
			key:       "/domains/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid namespace: expected features, got domains",
		},
		{
			name:      "missing feature path",
			key:       "/features/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
		{
			name:      "invalid CID format",
			key:       "/features/llm/invalid-cid/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid CID format",
		},
		{
			name:      "invalid value CID",
			key:       "/features/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte("invalid-cid-value"),
			wantError: true,
			errorMsg:  "invalid CID in value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.key, tt.value)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidators_Select(t *testing.T) {
	tests := []struct {
		name      string
		validator interface {
			Select(string, [][]byte) (int, error)
		}
		key       string
		values    [][]byte
		wantIndex int
		wantError bool
		errorMsg  string
	}{
		{
			name:      "skills validator - select first valid value",
			validator: &SkillValidator{},
			key:       "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			values: [][]byte{
				[]byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
				[]byte("invalid-cid"),
			},
			wantIndex: 0,
			wantError: false,
		},
		{
			name:      "domains validator - select first valid from multiple",
			validator: &DomainValidator{},
			key:       "/domains/ai/machine-learning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2",
			values: [][]byte{
				[]byte("invalid-cid"),
				[]byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
				[]byte(""),
			},
			wantIndex: 1,
			wantError: false,
		},
		{
			name:      "features validator - no valid values",
			validator: &FeatureValidator{},
			key:       "/features/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
			values: [][]byte{
				[]byte("invalid-cid-1"),
				[]byte("invalid-cid-2"),
			},
			wantIndex: -1,
			wantError: true,
			errorMsg:  "no valid values found",
		},
		{
			name:      "empty values slice",
			validator: &SkillValidator{},
			key:       "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			values:    [][]byte{},
			wantIndex: -1,
			wantError: true,
			errorMsg:  "no values to select from",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index, err := tt.validator.Select(tt.key, tt.values)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, tt.wantIndex, index)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantIndex, index)
			}
		})
	}
}

func TestNamespaceType(t *testing.T) {
	// Test String() method
	assert.Equal(t, "skills", NamespaceSkills.String())
	assert.Equal(t, "domains", NamespaceDomains.String())
	assert.Equal(t, "features", NamespaceFeatures.String())

	// Test Prefix() method
	assert.Equal(t, "/skills/", NamespaceSkills.Prefix())
	assert.Equal(t, "/domains/", NamespaceDomains.Prefix())
	assert.Equal(t, "/features/", NamespaceFeatures.Prefix())

	// Test IsValid() method
	assert.True(t, NamespaceSkills.IsValid())
	assert.True(t, NamespaceDomains.IsValid())
	assert.True(t, NamespaceFeatures.IsValid())
	assert.False(t, NamespaceType("invalid").IsValid())

	// Test ParseNamespace() function
	ns, valid := ParseNamespace("skills")
	assert.True(t, valid)
	assert.Equal(t, NamespaceSkills, ns)

	ns, valid = ParseNamespace("invalid")
	assert.False(t, valid)
	assert.Equal(t, NamespaceType(""), ns)

	// Test AllNamespaces() function
	all := AllNamespaces()
	assert.Len(t, all, 4)
	assert.Contains(t, all, NamespaceSkills)
	assert.Contains(t, all, NamespaceDomains)
	assert.Contains(t, all, NamespaceFeatures)
	assert.Contains(t, all, NamespaceLocators)

	// Test IsValidNamespaceKey() function
	assert.True(t, IsValidNamespaceKey("/skills/golang/CID123"))
	assert.True(t, IsValidNamespaceKey("/domains/web/CID123"))
	assert.True(t, IsValidNamespaceKey("/features/chat/CID123"))
	assert.True(t, IsValidNamespaceKey("/locators/docker-image/CID123"))
	assert.False(t, IsValidNamespaceKey("/invalid/test/CID123"))
	assert.False(t, IsValidNamespaceKey("/records/CID123"))
	assert.False(t, IsValidNamespaceKey("skills/golang/CID123")) // missing leading slash

	// Test GetNamespaceFromKey() function
	ns, found := GetNamespaceFromKey("/skills/golang/CID123")
	assert.True(t, found)
	assert.Equal(t, NamespaceSkills, ns)

	ns, found = GetNamespaceFromKey("/domains/web/CID123")
	assert.True(t, found)
	assert.Equal(t, NamespaceDomains, ns)

	ns, found = GetNamespaceFromKey("/features/chat/CID123")
	assert.True(t, found)
	assert.Equal(t, NamespaceFeatures, ns)

	ns, found = GetNamespaceFromKey("/invalid/test/CID123")
	assert.False(t, found)
	assert.Equal(t, NamespaceType(""), ns)
}

func TestCreateLabelValidators(t *testing.T) {
	validators := CreateLabelValidators()

	// Test that all expected validators are created
	assert.Len(t, validators, 4)
	assert.Contains(t, validators, NamespaceSkills.String())
	assert.Contains(t, validators, NamespaceDomains.String())
	assert.Contains(t, validators, NamespaceFeatures.String())
	assert.Contains(t, validators, NamespaceLocators.String())

	// Test that validators are of correct types
	assert.IsType(t, &SkillValidator{}, validators[NamespaceSkills.String()])
	assert.IsType(t, &DomainValidator{}, validators[NamespaceDomains.String()])
	assert.IsType(t, &FeatureValidator{}, validators[NamespaceFeatures.String()])
	assert.IsType(t, &LocatorValidator{}, validators[NamespaceLocators.String()])
}

func TestValidateLabelKey(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid skills key",
			key:       "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			wantError: false,
		},
		{
			name:      "valid domains key",
			key:       "/domains/ai/machine-learning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			wantError: false,
		},
		{
			name:      "valid features key",
			key:       "/features/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			wantError: false,
		},
		{
			name:      "invalid format - too few parts",
			key:       "/skills/programming",
			wantError: true,
			errorMsg:  "invalid key format: expected /<namespace>/<label_path>/<cid>",
		},
		{
			name:      "unsupported namespace",
			key:       "/unknown/path/value/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			wantError: true,
			errorMsg:  "unsupported namespace: unknown",
		},
		{
			name:      "missing CID",
			key:       "/skills/programming/golang/",
			wantError: true,
			errorMsg:  "missing CID in key",
		},
		{
			name:      "invalid CID format",
			key:       "/skills/programming/golang/invalid-cid-format",
			wantError: true,
			errorMsg:  "invalid CID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLabelKey(tt.key)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFormatLabelKey(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		cid      string
		expected string
	}{
		{
			name:     "label with leading slash",
			label:    "/skills/programming/golang",
			cid:      "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			expected: "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
		},
		{
			name:     "label without leading slash",
			label:    "skills/programming/golang",
			cid:      "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			expected: "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
		},
		{
			name:     "label with trailing slash",
			label:    "/domains/ai/machine-learning/",
			cid:      "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			expected: "/domains/ai/machine-learning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
		},
		{
			name:     "single component label",
			label:    "/features/llm",
			cid:      "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			expected: "/features/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLabelKey(tt.label, tt.cid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBaseValidator_validateKeyFormat(t *testing.T) {
	validator := &BaseValidator{}

	tests := []struct {
		name              string
		key               string
		expectedNamespace string
		wantError         bool
		errorMsg          string
		expectedParts     []string
	}{
		{
			name:              "valid key format",
			key:               "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			expectedNamespace: NamespaceSkills.String(),
			wantError:         false,
			expectedParts:     []string{"", "skills", "programming", "golang", "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku", "Peer1"},
		},
		{
			name:              "invalid format - too few parts",
			key:               "/skills/programming",
			expectedNamespace: NamespaceSkills.String(),
			wantError:         true,
			errorMsg:          "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
		{
			name:              "wrong namespace",
			key:               "/domains/ai/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			expectedNamespace: NamespaceSkills.String(),
			wantError:         true,
			errorMsg:          "invalid namespace: expected skills, got domains",
		},
		{
			name:              "invalid CID",
			key:               "/skills/programming/golang/invalid-cid/Peer1",
			expectedNamespace: NamespaceSkills.String(),
			wantError:         true,
			errorMsg:          "invalid CID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := validator.validateKeyFormat(tt.key, tt.expectedNamespace)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, parts)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedParts, parts)
			}
		})
	}
}

func TestBaseValidator_validateValue(t *testing.T) {
	validator := &BaseValidator{}

	tests := []struct {
		name      string
		value     []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "empty value",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid CID value",
			value:     []byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
			wantError: false,
		},
		{
			name:      "invalid CID value",
			value:     []byte("invalid-cid-format"),
			wantError: true,
			errorMsg:  "invalid CID in value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateValue(tt.value)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Benchmark tests to ensure validators perform well.
func BenchmarkSkillValidator_Validate(b *testing.B) {
	validator := &SkillValidator{}
	key := "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1"
	value := []byte{}

	b.ResetTimer()

	for range b.N {
		_ = validator.Validate(key, value)
	}
}

func BenchmarkDomainValidator_Validate(b *testing.B) {
	validator := &DomainValidator{}
	key := "/domains/ai/machine-learning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2"
	value := []byte{}

	b.ResetTimer()

	for range b.N {
		_ = validator.Validate(key, value)
	}
}

func BenchmarkFeatureValidator_Validate(b *testing.B) {
	validator := &FeatureValidator{}
	key := "/features/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3"
	value := []byte{}

	b.ResetTimer()

	for range b.N {
		_ = validator.Validate(key, value)
	}
}

func TestExtractCIDFromLabelKey(t *testing.T) {
	tests := []struct {
		name      string
		labelKey  string
		wantCID   string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid skills key",
			labelKey:  "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			wantCID:   "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			wantError: false,
		},
		{
			name:      "valid domains key",
			labelKey:  "/domains/ai/machine-learning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2",
			wantCID:   "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			wantError: false,
		},
		{
			name:      "valid features key",
			labelKey:  "/features/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
			wantCID:   "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			wantError: false,
		},
		{
			name:      "invalid format - too few parts",
			labelKey:  "/skills/programming",
			wantError: true,
			errorMsg:  "invalid enhanced key format",
		},
		{
			name:      "invalid namespace",
			labelKey:  "/unknown/test/value/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			wantError: true,
			errorMsg:  "invalid namespace",
		},
		{
			name:      "invalid CID format",
			labelKey:  "/skills/programming/golang/invalid-cid/Peer1",
			wantError: true,
			errorMsg:  "invalid CID format",
		},
		{
			name:      "missing CID",
			labelKey:  "/skills/programming/golang//Peer1",
			wantError: true,
			errorMsg:  "missing CID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid, err := ExtractCIDFromLabelKey(tt.labelKey)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Equal(t, "", cid)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantCID, cid)
			}
		})
	}
}

func BenchmarkFormatLabelKey(b *testing.B) {
	label := "/skills/programming/golang"
	cid := "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"

	b.ResetTimer()

	for range b.N {
		_ = FormatLabelKey(label, cid)
	}
}

func BenchmarkExtractCIDFromLabelKey(b *testing.B) {
	labelKey := "/skills/programming/golang/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1"

	b.ResetTimer()

	for range b.N {
		_, _ = ExtractCIDFromLabelKey(labelKey)
	}
}
