// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package python

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	pipFileNoVersion         = "./testdata/pipfile-no-version/Pipfile"
	pipFileWithVersion       = "./testdata/pipfile-version/Pipfile"
	pyprojectNoVersion       = "./testdata/pyproject-no-version/pyproject.toml"
	pyprojectPoetryVersion   = "./testdata/pyproject-poetry/pyproject.toml"
	pyprojectStandardVersion = "./testdata/pyproject-standard/pyproject.toml"
	setupPyNoVersion         = "./testdata/setup-no-version/setup.py"
	setupPyWithVersion       = "./testdata/setup-version/setup.py"
	unsupported              = "./testdata/unsupported/unsupported"
)

func TestGetRuntimeInfo(t *testing.T) {
	tests := []struct {
		name            string
		file            string
		wantErr         bool
		errorContains   string
		expectedVersion string
	}{
		{
			name:          "pipfile-no-version",
			file:          pipFileNoVersion,
			wantErr:       true,
			errorContains: errNoVersion.Error(),
		},
		{
			name:            "pipfile-with-version",
			file:            pipFileWithVersion,
			wantErr:         false,
			expectedVersion: ">=3.9",
		},
		{
			name:          "pyproject-no-version",
			file:          pyprojectNoVersion,
			wantErr:       true,
			errorContains: errNoVersion.Error(),
		},
		{
			name:            "pyproject-poetry-version",
			file:            pyprojectPoetryVersion,
			wantErr:         false,
			expectedVersion: ">=3.11",
		},
		{
			name:            "pyproject-standard-version",
			file:            pyprojectStandardVersion,
			wantErr:         false,
			expectedVersion: ">=3.11",
		},
		{
			name:          "setup-no-version",
			file:          setupPyNoVersion,
			wantErr:       true,
			errorContains: errNoVersion.Error(),
		},
		{
			name:            "setup-with-version",
			file:            setupPyWithVersion,
			wantErr:         false,
			expectedVersion: ">=3.6",
		},
		{
			name:          "unsupported",
			file:          unsupported,
			wantErr:       true,
			errorContains: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret, err := getRuntimeInfo(tt.file)

			if tt.expectedVersion != "" && err == nil {
				assert.Equal(t, tt.expectedVersion, ret.Version)
				assert.Equal(t, "python", ret.Language)
			}

			if tt.wantErr && err == nil {
				t.Errorf("parsePipfile() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("parsePipfile() error = %v, wantErr %v", err, tt.errorContains)
			}
		})
	}
}
