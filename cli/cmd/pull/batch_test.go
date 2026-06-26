// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rec(name, version string) *corev1.Record {
	return corev1.New(&oasfv1alpha1.Record{Name: name, Version: version})
}

func TestSanitizeName(t *testing.T) {
	assert.Equal(t, "cisco.com-my-agent", sanitizeName("cisco.com/my-agent"))
	assert.Equal(t, "a-b-c", sanitizeName("a:b c"))
}

func TestLatestByNameKeepsHighestSemver(t *testing.T) {
	records := []*corev1.Record{
		rec("agent-a", "1.0.0"),
		rec("agent-a", "2.1.0"),
		rec("agent-b", "0.1.0"),
		rec("agent-a", "1.5.0"),
	}

	latest := latestByName(records)
	require.Len(t, latest, 2)

	byName := map[string]string{}
	for _, r := range latest {
		byName[r.GetName()] = r.GetVersion()
	}

	assert.Equal(t, "2.1.0", byName["agent-a"])
	assert.Equal(t, "0.1.0", byName["agent-b"])
}

func TestBatchFileNameUsesNameAndVersion(t *testing.T) {
	seen := map[string]int{}

	assert.Equal(t, "my-agent", batchFileName(rec("my-agent", "1.0.0"), 0, seen, false))
	assert.Equal(t, "my-agent-1.0.0", batchFileName(rec("my-agent", "1.0.0"), 0, seen, true))
}

func TestBatchFileNameFallsBackToIndex(t *testing.T) {
	seen := map[string]int{}
	assert.Equal(t, "record_3", batchFileName(rec("", ""), 3, seen, false))
}

func TestValidatePullInvocation(t *testing.T) {
	tests := []struct {
		name       string
		hasArg     bool
		outputFile string
		outputDir  string
		hasFilters bool
		wantErr    bool
	}{
		{name: "single arg to stdout", hasArg: true, wantErr: false},
		{name: "single arg to file", hasArg: true, outputFile: "/tmp/x.json", wantErr: false},
		{name: "batch dir with filters", outputDir: "/tmp/out", hasFilters: true, wantErr: false},
		{name: "no arg no dir", wantErr: true},
		{name: "arg and dir together", hasArg: true, outputDir: "/tmp/out", hasFilters: true, wantErr: true},
		{name: "file and dir together", outputFile: "/tmp/x.json", outputDir: "/tmp/out", hasFilters: true, wantErr: true},
		{name: "dir without filters", outputDir: "/tmp/out", wantErr: true},
		{name: "file without arg", outputFile: "/tmp/x.json", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePullInvocation(tt.hasArg, tt.outputFile, tt.outputDir, tt.hasFilters)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
