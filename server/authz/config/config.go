package config

var (
	// DefaultSocketPath is the default path for the SPIFFE Workload API socket
	DefaultSocketPath = "/tmp/agent.sock"

	// DefaultTrustDomain is the default SPIFFE trust domain for the AuthZ service
	DefaultTrustDomain = "example.com"
)

// Config contains configuration for AuthZ services
type Config struct {
	SocketPath  string `json:"socket_path,omitempty" mapstructure:"socket_path"`
	TrustDomain string `json:"trust_domain,omitempty" mapstructure:"trust_domain"`
}
