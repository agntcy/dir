// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	"github.com/agntcy/dir/discovery/pkg/processor/a2a"
	processor "github.com/agntcy/dir/discovery/pkg/processor/config"
	"github.com/agntcy/dir/discovery/pkg/processor/oasf"
	runtime "github.com/agntcy/dir/discovery/pkg/runtime/config"
	"github.com/agntcy/dir/discovery/pkg/runtime/docker"
	"github.com/agntcy/dir/discovery/pkg/runtime/k8s"
	storage "github.com/agntcy/dir/discovery/pkg/storage/config"
	"github.com/agntcy/dir/discovery/pkg/storage/etcd"
	"github.com/alecthomas/assert/v2"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		Name           string
		EnvVars        map[string]string
		ExpectedConfig *Config
	}{
		{
			Name: "Custom config",
			EnvVars: map[string]string{
				// server
				"DISCOVERY_SERVER_HOST": "9090",
				"DISCOVERY_SERVER_PORT": "9090",

				// runtime
				"DISCOVERY_RUNTIME_TYPE":                      "kubernetes",
				"DISCOVERY_RUNTIME_DOCKER_HOST":               "tcp://docker.local:2375",
				"DISCOVERY_RUNTIME_DOCKER_LABEL_KEY":          "env",
				"DISCOVERY_RUNTIME_DOCKER_LABEL_VALUE":        "prod",
				"DISCOVERY_RUNTIME_KUBERNETES_KUBECONFIG":     "/path/to/kubeconfig",
				"DISCOVERY_RUNTIME_KUBERNETES_NAMESPACE":      "production",
				"DISCOVERY_RUNTIME_KUBERNETES_IN_CLUSTER":     "true",
				"DISCOVERY_RUNTIME_KUBERNETES_LABEL_KEY":      "env",
				"DISCOVERY_RUNTIME_KUBERNETES_LABEL_VALUE":    "prod",
				"DISCOVERY_RUNTIME_KUBERNETES_WATCH_SERVICES": "true",

				// storage
				"DISCOVERY_STORAGE_TYPE":                  "etcd",
				"DISCOVERY_STORAGE_ETCD_HOST":             "etcd.local",
				"DISCOVERY_STORAGE_ETCD_PORT":             "1234",
				"DISCOVERY_STORAGE_ETCD_USERNAME":         "user",
				"DISCOVERY_STORAGE_ETCD_PASSWORD":         "pass",
				"DISCOVERY_STORAGE_ETCD_DIAL_TIMEOUT":     "10s",
				"DISCOVERY_STORAGE_ETCD_WORKLOADS_PREFIX": "/custom/workloads/",

				// processor
				"DISCOVERY_PROCESSOR_WORKERS":         "8",
				"DISCOVERY_PROCESSOR_A2A_ENABLED":     "true",
				"DISCOVERY_PROCESSOR_A2A_TIMEOUT":     "20s",
				"DISCOVERY_PROCESSOR_A2A_PATHS":       "/a2a,/a2a2",
				"DISCOVERY_PROCESSOR_A2A_LABEL_KEY":   "a2a_key",
				"DISCOVERY_PROCESSOR_A2A_LABEL_VALUE": "a2a_value",
				"DISCOVERY_PROCESSOR_OASF_ENABLED":    "false",
				"DISCOVERY_PROCESSOR_OASF_TIMEOUT":    "15s",
				"DISCOVERY_PROCESSOR_OASF_LABEL_KEY":  "oasf_key",
			},
			ExpectedConfig: &Config{
				Server: ServerConfig{
					Host: "9090",
					Port: 9090,
				},
				Runtime: runtime.Config{
					Type: "kubernetes",
					Docker: docker.Config{
						Host:       "tcp://docker.local:2375",
						LabelKey:   "env",
						LabelValue: "prod",
					},
					Kubernetes: k8s.Config{
						Kubeconfig:    "/path/to/kubeconfig",
						Namespace:     "production",
						InCluster:     true,
						LabelKey:      "env",
						LabelValue:    "prod",
						WatchServices: true,
					},
				},
				Storage: storage.Config{
					StorageType: "etcd",
					Etcd: etcd.Config{
						Host:            "etcd.local",
						Port:            1234,
						Username:        "user",
						Password:        "pass",
						DialTimeout:     10 * time.Second,
						WorkloadsPrefix: "/custom/workloads/",
					},
				},
				Processor: processor.Config{
					Workers: 8,
					A2A: a2a.Config{
						Enabled:    true,
						Timeout:    20 * time.Second,
						Paths:      []string{"/a2a", "/a2a2"},
						LabelKey:   "a2a_key",
						LabelValue: "a2a_value",
					},
					OASF: oasf.Config{
						Enabled:  false,
						Timeout:  15 * time.Second,
						LabelKey: "oasf_key",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			for k, v := range test.EnvVars {
				t.Setenv(k, v)
			}

			config, err := LoadConfig()
			assert.NoError(t, err)
			assert.Equal(t, *config, *test.ExpectedConfig)
		})
	}
}
