// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package corev1alpha1

import (
	"fmt"
)

func (agent *Agent) ObjectMeta() (*ObjectMeta, error) {
	return &ObjectMeta{
		Type:        ObjectType_OBJECT_TYPE_AGENT,
		Name:        fmt.Sprintf("%s:%s", agent.GetName(), agent.GetVersion()),
		Annotations: agent.GetAnnotations(),
		Digest:      agent.GetDigest(),
	}, nil
}

func (locator *Locator) ObjectMeta() (*ObjectMeta, error) {
	return &ObjectMeta{
		Type: ObjectType_OBJECT_TYPE_LOCATOR,
		Name: fmt.Sprintf("%s:%s:%s",
			locator.GetType().String(),
			locator.GetName(),
			locator.GetSource().GetUrl(), // url may contain ":" delimeter, so careful when parsing back
		),
		Annotations: locator.GetAnnotations(),
		Digest:      locator.GetDigest(),
	}, nil
}

func (extension *Extension) ObjectMeta() (*ObjectMeta, error) {
	return &ObjectMeta{
		Type:        ObjectType_OBJECT_TYPE_EXTENSION,
		Name:        fmt.Sprintf("%s:%s", extension.GetName(), extension.GetVersion()),
		Annotations: extension.GetAnnotations(),
		Digest:      extension.GetDigest(),
	}, nil
}
