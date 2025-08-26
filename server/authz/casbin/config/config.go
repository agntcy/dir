package config

type Config struct {
	// Path to the Casbin model configuration file
	ModelPath string `json:"model_path,omitempty" mapstructure:"model_path"`

	// Path to the Casbin policy file
	PolicyPath string `json:"policy_path,omitempty" mapstructure:"policy_path"`
}
