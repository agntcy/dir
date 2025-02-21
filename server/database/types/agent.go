// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

type Agent struct {
	Model

	Name    string `json:"name,omitempty" mapstructure:"name"`
	Version string `json:"version,omitempty" mapstructure:"version"`
	Digest  string `json:"digest,omitempty" mapstructure:"digest"`
}
