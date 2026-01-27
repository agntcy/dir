package k8s

// Config holds Kubernetes runtime configuration.
type Config struct {
	// Kubeconfig is the path to kubeconfig file (empty for in-cluster).
	Kubeconfig string `json:"kubeconfig,omitempty" mapstructure:"kubeconfig"`

	// Namespace to watch (empty for all namespaces).
	Namespace string `json:"namespace,omitempty" mapstructure:"namespace"`

	// InCluster uses in-cluster Kubernetes configuration.
	InCluster bool `json:"in_cluster,omitempty" mapstructure:"in_cluster"`

	// LabelKey is the label key for discoverable pods.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the label value for discoverable pods.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`

	// WatchServices enables watching Kubernetes services.
	WatchServices bool `json:"watch_services,omitempty" mapstructure:"watch_services"`
}
