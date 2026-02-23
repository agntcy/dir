// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package etcd

import (
	"strconv"
	"time"
)

const (
	// DefaultWorkloadsPrefix is the default etcd key prefix for workloads.
	DefaultWorkloadsPrefix = "/discovery/workloads/"

	// DefaultDialTimeout is the default timeout for connecting to etcd.
	DefaultDialTimeout = 5 * time.Second

	// DefaultPort is the default etcd server port.
	DefaultPort = 2379

	// DefaultHost is the default etcd server hostname.
	DefaultHost = "localhost"
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
	//nolint:gosec // G117: intentional config field for etcd auth
	Password string `json:"password,omitempty" mapstructure:"password"`

	// DialTimeout is the timeout for connecting to etcd.
	DialTimeout time.Duration `json:"dial_timeout,omitempty" mapstructure:"dial_timeout"`

	// WorkloadsPrefix is the etcd key prefix for workloads.
	WorkloadsPrefix string `json:"workloads_prefix,omitempty" mapstructure:"workloads_prefix"`
}

// Endpoints returns the etcd endpoint URLs.
func (c *Config) Endpoints() []string {
	return []string{c.Host + ":" + strconv.Itoa(c.Port)}
}
