// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	routingv1 "github.com/agntcy/dir/api/routing/v1"
)

type PublicationObject interface {
	GetID() string
	GetRequest() *routingv1.PublishRequest
	GetStatus() routingv1.PublicationStatus
	GetCreatedTime() string
	GetLastUpdateTime() string
}
