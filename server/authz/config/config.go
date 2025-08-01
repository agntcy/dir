package config

// Config contains configuration for AuthZ services
type Config struct {
	SocketPath  string `json:"socket_path,omitempty" mapstructure:"socket_path"`
	TrustDomain string `json:"trust_domain,omitempty" mapstructure:"trust_domain"`
}
