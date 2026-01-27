package docker

// Config holds Docker runtime configuration.
type Config struct {
	// Host is the Docker daemon socket path.
	Host string `json:"host,omitempty" mapstructure:"host"`

	// LabelKey is the Docker label key to filter containers.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the Docker label value to filter containers.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`
}
