// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package localfs

const (
	DefaultDir = "/tmp"
)

type Config struct {
	Dir string `json:"localfs_dir,omitempty" mapstructure:"localfs_dir"`
}
