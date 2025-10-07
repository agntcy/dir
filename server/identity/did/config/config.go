package config

type Config struct {
	Port        int    `json:"port" yaml:"port"`
	DatabaseURL string `json:"database_url" yaml:"database_url"`
	LogLevel    string `json:"log_level" yaml:"log_level"`
}
