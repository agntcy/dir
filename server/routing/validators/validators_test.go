// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"strings"
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Add utility functions for testing.
func GetLabelTypeFromKey(key string) (types.LabelType, bool) {
	for _, labelType := range types.AllLabelTypes() {
		if strings.HasPrefix(key, labelType.Prefix()) {
			return labelType, true
		}
	}

	return types.LabelTypeUnknown, false
}

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
		{
			name:      "missing CID",
			key:       "/domains/ai//Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "missing CID in key",
		},
		{
			name:      "missing PeerID",
			key:       "/domains/ai/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
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
func TestModuleValidator_Validate(t *testing.T) {
	validator := &ModuleValidator{}

	tests := []struct {
		name      string
		key       string
		value     []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid modules key with single module",
			key:       "/modules/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid modules key with nested module path",
			key:       "/modules/ai/reasoning/logical/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid modules key with value",
			key:       "/modules/search/semantic/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
			value:     []byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
			wantError: false,
		},
		{
			name:      "invalid namespace",
			key:       "/domains/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid namespace: expected modules, got domains",
		},
		{
			name:      "missing module path",
			key:       "/modules/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
		{
			name:      "invalid CID format",
			key:       "/modules/llm/invalid-cid/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid CID format",
		},
		{
			name:      "invalid value CID",
			key:       "/modules/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte("invalid-cid-value"),
			wantError: true,
			errorMsg:  "invalid CID in value",
		},
		{
			name:      "missing CID",
			key:       "/modules/llm//Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "missing CID in key",
		},
		{
			name:      "missing PeerID",
			key:       "/modules/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
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
func TestLocatorValidator_Validate(t *testing.T) {
	validator := &LocatorValidator{}

	tests := []struct {
		name      string
		key       string
		value     []byte
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid locators key with single locator type",
			key:       "/locators/docker-image/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid locators key with nested locator path",
			key:       "/locators/container/docker/alpine/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer2",
			value:     []byte{},
			wantError: false,
		},
		{
			name:      "valid locators key with value",
			key:       "/locators/npm-package/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
			value:     []byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
			wantError: false,
		},
		{
			name:      "invalid namespace",
			key:       "/modules/docker-image/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid namespace: expected locators, got modules",
		},
		{
			name:      "missing locator type",
			key:       "/locators/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
		{
			name:      "invalid CID format",
			key:       "/locators/docker-image/invalid-cid/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "invalid CID format",
		},
		{
			name:      "invalid value CID",
			key:       "/locators/docker-image/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte("invalid-cid-value"),
			wantError: true,
			errorMsg:  "invalid CID in value",
		},
		{
			name:      "empty locator path component",
			key:       "/locators//docker-image/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "locator path component cannot be empty at position 1",
		},
		{
			name:      "empty locator path component in middle",
			key:       "/locators/container//alpine/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "locator path component cannot be empty at position 2",
		},
		{
			name:      "missing CID",
			key:       "/locators/docker-image//Peer1",
			value:     []byte{},
			wantError: true,
			errorMsg:  "missing CID in key",
		},
		{
			name:      "missing PeerID",
			key:       "/locators/docker-image/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
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
			name:      "modules validator - no valid values",
			validator: &ModuleValidator{},
			key:       "/modules/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
			values: [][]byte{
				[]byte("invalid-cid-1"),
				[]byte("invalid-cid-2"),
			},
			wantIndex: -1,
			wantError: true,
			errorMsg:  "no valid values found",
		},
		{
			name:      "locators validator - select first valid value",
			validator: &LocatorValidator{},
			key:       "/locators/docker-image/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			values: [][]byte{
				[]byte("bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku"),
				[]byte("invalid-cid"),
			},
			wantIndex: 0,
			wantError: false,
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

func TestLabelTypeIntegration(t *testing.T) {
	// Test that LabelType works correctly with validators
	// Test String() method
	assert.Equal(t, "skills", types.LabelTypeSkill.String())
	assert.Equal(t, "domains", types.LabelTypeDomain.String())
	assert.Equal(t, "modules", types.LabelTypeModule.String())
	assert.Equal(t, "locators", types.LabelTypeLocator.String())

	// Test Prefix() method
	assert.Equal(t, "/skills/", types.LabelTypeSkill.Prefix())
	assert.Equal(t, "/domains/", types.LabelTypeDomain.Prefix())
	assert.Equal(t, "/modules/", types.LabelTypeModule.Prefix())
	assert.Equal(t, "/locators/", types.LabelTypeLocator.Prefix())

	// Test IsValid() method
	assert.True(t, types.LabelTypeSkill.IsValid())
	assert.True(t, types.LabelTypeDomain.IsValid())
	assert.True(t, types.LabelTypeModule.IsValid())
	assert.True(t, types.LabelTypeLocator.IsValid())
	assert.False(t, types.LabelType("invalid").IsValid())

	// Test ParseLabelType() function
	lt, valid := types.ParseLabelType("skills")
	assert.True(t, valid)
	assert.Equal(t, types.LabelTypeSkill, lt)

	lt, valid = types.ParseLabelType("invalid")
	assert.False(t, valid)
	assert.Equal(t, types.LabelTypeUnknown, lt)

	// Test AllLabelTypes() function
	all := types.AllLabelTypes()
	assert.Len(t, all, 4)
	assert.Contains(t, all, types.LabelTypeSkill)
	assert.Contains(t, all, types.LabelTypeDomain)
	assert.Contains(t, all, types.LabelTypeModule)
	assert.Contains(t, all, types.LabelTypeLocator)

	// Test IsValidLabelKey() function
	assert.True(t, IsValidLabelKey("/skills/golang/CID123"))
	assert.True(t, IsValidLabelKey("/domains/web/CID123"))
	assert.True(t, IsValidLabelKey("/modules/chat/CID123"))
	assert.True(t, IsValidLabelKey("/locators/docker-image/CID123"))
	assert.False(t, IsValidLabelKey("/invalid/test/CID123"))
	assert.False(t, IsValidLabelKey("/records/CID123"))
	assert.False(t, IsValidLabelKey("skills/golang/CID123")) // missing leading slash

	// Test GetLabelTypeFromKey() function
	lt, found := GetLabelTypeFromKey("/skills/golang/CID123")
	assert.True(t, found)
	assert.Equal(t, types.LabelTypeSkill, lt)

	lt, found = GetLabelTypeFromKey("/domains/web/CID123")
	assert.True(t, found)
	assert.Equal(t, types.LabelTypeDomain, lt)

	lt, found = GetLabelTypeFromKey("/modules/chat/CID123")
	assert.True(t, found)
	assert.Equal(t, types.LabelTypeModule, lt)

	lt, found = GetLabelTypeFromKey("/invalid/test/CID123")
	assert.False(t, found)
	assert.Equal(t, types.LabelTypeUnknown, lt)
}

func TestCreateLabelValidators(t *testing.T) {
	validators := CreateLabelValidators()

	// Test that all expected validators are created
	assert.Len(t, validators, 4)
	assert.Contains(t, validators, types.LabelTypeSkill.String())
	assert.Contains(t, validators, types.LabelTypeDomain.String())
	assert.Contains(t, validators, types.LabelTypeModule.String())
	assert.Contains(t, validators, types.LabelTypeLocator.String())

	// Test that validators are of correct types
	assert.IsType(t, &SkillValidator{}, validators[types.LabelTypeSkill.String()])
	assert.IsType(t, &DomainValidator{}, validators[types.LabelTypeDomain.String()])
	assert.IsType(t, &ModuleValidator{}, validators[types.LabelTypeModule.String()])
	assert.IsType(t, &LocatorValidator{}, validators[types.LabelTypeLocator.String()])
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
			name:      "valid modules key",
			key:       "/modules/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
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
			label:    "/modules/llm",
			cid:      "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
			expected: "/modules/llm/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku",
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
			expectedNamespace: types.LabelTypeSkill.String(),
			wantError:         false,
			expectedParts:     []string{"", "skills", "programming", "golang", "bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku", "Peer1"},
		},
		{
			name:              "invalid format - too few parts",
			key:               "/skills/programming",
			expectedNamespace: types.LabelTypeSkill.String(),
			wantError:         true,
			errorMsg:          "invalid key format: expected /<namespace>/<specific_path>/<cid>/<peer_id>",
		},
		{
			name:              "wrong namespace",
			key:               "/domains/ai/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1",
			expectedNamespace: types.LabelTypeSkill.String(),
			wantError:         true,
			errorMsg:          "invalid namespace: expected skills, got domains",
		},
		{
			name:              "invalid CID",
			key:               "/skills/programming/golang/invalid-cid/Peer1",
			expectedNamespace: types.LabelTypeSkill.String(),
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

func BenchmarkModuleValidator_Validate(b *testing.B) {
	validator := &ModuleValidator{}
	key := "/modules/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3"
	value := []byte{}

	b.ResetTimer()

	for range b.N {
		_ = validator.Validate(key, value)
	}
}

func BenchmarkLocatorValidator_Validate(b *testing.B) {
	validator := &LocatorValidator{}
	key := "/locators/docker-image/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer1"
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
			name:      "valid modules key",
			labelKey:  "/modules/llm/reasoning/bafkreihdwdcefgh4dqkjv67uzcmw7ojee6xedzdetojuzjevtenxquvyku/Peer3",
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
				assert.Empty(t, cid)
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
