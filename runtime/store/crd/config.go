// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package crd

import (
	"time"
)

var (
	// DefaultNamespace is the default namespace to store workloads in.
	DefaultNamespace = "default"

	// DefaultResyncPeriod is how often to resync the cache from the API server.
	DefaultResyncPeriod = 30 * time.Second
)

// Config holds Kubernetes CRD storage configuration.
type Config struct {
	// Namespace is the namespace to store workloads in (default: default).
	Namespace string `json:"namespace,omitempty" mapstructure:"namespace"`

	// Kubeconfig path for out-of-cluster access. If empty, in-cluster config is used.
	Kubeconfig string `json:"kubeconfig,omitempty" mapstructure:"kubeconfig"`

	// ResyncPeriod is how often to resync the cache from the API server.
	ResyncPeriod time.Duration `json:"resync_period,omitempty" mapstructure:"resync_period"`
}
