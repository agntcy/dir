// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"fmt"
)

func (agent *Agent) ObjectMeta() (*ObjectMeta, error) {
	return &ObjectMeta{
		Type:        ObjectType_OBJECT_TYPE_AGENT,
		Name:        fmt.Sprintf("%s:%s", agent.Name, agent.Version),
		Annotations: agent.Annotations,
		Digest:      agent.Digest,
	}, nil
}

func (locator *Locator) ObjectMeta() (*ObjectMeta, error) {
	return &ObjectMeta{
		Type: ObjectType_OBJECT_TYPE_LOCATOR,
		Name: fmt.Sprintf("%s:%s:%s",
			locator.Type.String(),
			locator.Name,
			locator.Source.Url, // url may contain ":" delimeter, so careful when parsing back
		),
		Annotations: locator.Annotations,
		Digest:      locator.Digest,
	}, nil
}

func (extension *Extension) ObjectMeta() (*ObjectMeta, error) {
	return &ObjectMeta{
		Type:        ObjectType_OBJECT_TYPE_EXTENSION,
		Name:        fmt.Sprintf("%s:%s", extension.Name, extension.Version),
		Annotations: extension.Annotations,
		Digest:      extension.Digest,
	}, nil
}
