// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
