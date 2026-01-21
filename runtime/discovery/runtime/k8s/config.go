// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package k8s

const (
	// DefaultLabelKey is the default Kubernetes label key to filter pods.
	DefaultLabelKey = "org.agntcy/discover"

	// DefaultLabelValue is the default Kubernetes label value to filter pods.
	DefaultLabelValue = "true"

	// DefaultNamespace is the default namespace to watch (empty means all namespaces).
	DefaultNamespace = ""

	// DefaultKubeconfig is the default path to kubeconfig file (empty means in-cluster).
	DefaultKubeconfig = ""
)

// Config holds Kubernetes runtime configuration.
type Config struct {
	// Kubeconfig is the path to kubeconfig file (empty for in-cluster).
	Kubeconfig string `json:"kubeconfig,omitempty" mapstructure:"kubeconfig"`

	// Namespace to watch (empty for all namespaces).
	Namespace string `json:"namespace,omitempty" mapstructure:"namespace"`

	// LabelKey is the label key for discoverable pods.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the label value for discoverable pods.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`
}
