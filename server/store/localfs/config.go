// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package localfs

import "github.com/spf13/afero"

const (
	DefaultDir = "/tmp"
)

var DefaultFs = afero.NewOsFs()

type Config struct {
	Dir string `json:"localfs_dir,omitempty" mapstructure:"localfs_dir"`
}
