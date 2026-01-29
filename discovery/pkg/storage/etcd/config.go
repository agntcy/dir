package etcd

import (
	"strconv"
	"time"
)

// Config holds etcd connection configuration.
type Config struct {
	// Host is the etcd server hostname.
	Host string `json:"host,omitempty" mapstructure:"host"`

	// Port is the etcd server port.
	Port int `json:"port,omitempty" mapstructure:"port"`

	// Username for etcd authentication.
	Username string `json:"username,omitempty" mapstructure:"username"`

	// Password for etcd authentication.
	Password string `json:"password,omitempty" mapstructure:"password"`

	// DialTimeout is the timeout for connecting to etcd.
	DialTimeout time.Duration `json:"dial_timeout,omitempty" mapstructure:"dial_timeout"`

	// WorkloadsPrefix is the etcd key prefix for workloads.
	WorkloadsPrefix string `json:"workloads_prefix,omitempty" mapstructure:"workloads_prefix"`
}

func (c *Config) Endpoints() []string {
	return []string{c.Host + ":" + strconv.Itoa(c.Port)}
}
