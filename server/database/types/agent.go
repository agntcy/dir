// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

type Agent struct {
	Model `gorm:"embedded"`

	Name string `gorm:"not null"`
}
