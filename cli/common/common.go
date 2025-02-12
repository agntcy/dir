// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package common

import (
	apicore "github.com/agntcy/dir/api/core/v1alpha1"
)

func GetOwnerKeyAndSignatureForAgent(agent *apicore.Agent) (string, string) {
	for _, ext := range agent.Extensions {
		if ext.Name == "security" {
			return ext.Specs.Fields["owner_key"].GetStringValue(), ext.Specs.Fields["signature"].GetStringValue()
		}
	}

	return "", ""
}
