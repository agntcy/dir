// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLabelKeyIsDeterministic(t *testing.T) {
	t.Parallel()

	first, err := labelKey(types.Label("/skills/AI/ML"))
	require.NoError(t, err)

	second, err := labelKey(types.Label("/skills/AI/ML"))
	require.NoError(t, err)

	assert.Equal(t, first, second, "the same label must always resolve to the same DHT key")
}

func TestLabelKeyIgnoresTrailingSlashAndWhitespace(t *testing.T) {
	t.Parallel()

	canonical, err := labelKey(types.Label("/skills/AI/ML"))
	require.NoError(t, err)

	for _, variant := range []string{"/skills/AI/ML/", " /skills/AI/ML ", "\t/skills/AI/ML/\n"} {
		key, err := labelKey(types.Label(variant))
		require.NoError(t, err, variant)
		assert.Equal(t, canonical, key, "variant %q must match the canonical key", variant)
	}
}

func TestLabelKeySeparatesDistinctLabels(t *testing.T) {
	t.Parallel()

	skill, err := labelKey(types.Label("/skills/AI"))
	require.NoError(t, err)

	domain, err := labelKey(types.Label("/domains/AI"))
	require.NoError(t, err)

	assert.NotEqual(t, skill, domain, "the same value in different namespaces must not collide")
}

func TestLabelKeyRejectsEmptyLabel(t *testing.T) {
	t.Parallel()

	for _, empty := range []string{"", "   ", "/"} {
		_, err := labelKey(types.Label(empty))
		require.ErrorIs(t, err, errEmptyLabel, "expected an error for %q", empty)
	}
}

// A bare namespace matches everything, so it is not a usable key on either the
// publish or the search side.
func TestLabelKeyRejectsBareNamespace(t *testing.T) {
	t.Parallel()

	for _, namespace := range []string{"/skills", "/skills/", "/domains", "/modules", "/locators"} {
		_, err := labelKey(types.Label(namespace))
		require.ErrorIs(t, err, errBareNamespace, "expected an error for %q", namespace)
	}
}

func TestExpandLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		label string
		want  []types.Label
	}{
		{
			name:  "nested skill yields ancestors, closest first",
			label: "/skills/AI/ML/NLP",
			want:  []types.Label{"/skills/AI/ML/NLP", "/skills/AI/ML", "/skills/AI"},
		},
		{
			name:  "single segment has no ancestors",
			label: "/skills/AI",
			want:  []types.Label{"/skills/AI"},
		},
		{
			name:  "bare namespace expands to nothing",
			label: "/skills",
			want:  nil,
		},
		{
			name:  "namespace with trailing slash expands to nothing",
			label: "/skills/",
			want:  nil,
		},
		{
			name:  "locators expand like any other namespace",
			label: "/locators/docker-image",
			want:  []types.Label{"/locators/docker-image"},
		},
		{
			name:  "doubled slashes do not produce empty segments",
			label: "/domains/finance//banking",
			want:  []types.Label{"/domains/finance/banking", "/domains/finance"},
		},
		{
			name:  "unknown namespace is returned untouched",
			label: "/something/else/entirely",
			want:  []types.Label{"/something/else/entirely"},
		},
		{
			name:  "empty label expands to nothing",
			label: "",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, expandLabel(types.Label(tt.label)))
		})
	}
}

// Ancestor expansion must stop below the namespace root, however deep the
// label is.
func TestExpandLabelStopsBelowNamespaceRoot(t *testing.T) {
	t.Parallel()

	roots := []types.Label{"/skills", "/domains", "/modules", "/locators"}

	for _, label := range []string{"/skills/a/b/c/d", "/domains/finance/banking", "/modules/a/b"} {
		for _, got := range expandLabel(types.Label(label)) {
			assert.NotContains(t, roots, got, "expansion of %q must not reach a namespace root", label)
		}
	}
}

func TestExpandLabelsDeduplicatesSharedAncestors(t *testing.T) {
	t.Parallel()

	expanded := expandLabels([]types.Label{
		"/skills/AI/ML",
		"/skills/AI/NLP",
		"/skills/AI",
		"/domains/finance",
	})

	assert.Equal(t, []types.Label{
		"/skills/AI/ML",
		"/skills/AI",
		"/skills/AI/NLP",
		"/domains/finance",
	}, expanded)
}

func TestExpandLabelsHandlesEmptyInput(t *testing.T) {
	t.Parallel()

	assert.Empty(t, expandLabels(nil))
	assert.Empty(t, expandLabels([]types.Label{"", "/"}))
}
