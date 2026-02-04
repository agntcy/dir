// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	store "github.com/agntcy/dir/runtime/store/config"
	"github.com/agntcy/dir/runtime/store/crd"
	"github.com/agntcy/dir/runtime/store/etcd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		Name           string
		EnvVars        map[string]string
		ExpectedConfig *Config
	}{
		{
			Name: "Custom config with etcd store",
			EnvVars: map[string]string{
				"SERVER_HOST":                        "192.168.1.100",
				"SERVER_PORT":                        "9090",
				"SERVER_STORE_TYPE":                  "etcd",
				"SERVER_STORE_ETCD_HOST":             "etcd.example.com",
				"SERVER_STORE_ETCD_PORT":             "2380",
				"SERVER_STORE_ETCD_USERNAME":         "admin",
				"SERVER_STORE_ETCD_PASSWORD":         "secret",
				"SERVER_STORE_ETCD_DIAL_TIMEOUT":     "10s",
				"SERVER_STORE_ETCD_WORKLOADS_PREFIX": "/custom/workloads/",
				"SERVER_STORE_CRD_NAMESPACE":         "crd-namespace",
				"SERVER_STORE_CRD_KUBECONFIG":        "crd-kubeconfig",
				"SERVER_STORE_CRD_RESYNC_PERIOD":     "15s",
			},
			ExpectedConfig: &Config{
				Host: "192.168.1.100",
				Port: 9090,
				Store: store.Config{
					Type: etcd.StoreType,
					Etcd: etcd.Config{
						Host:            "etcd.example.com",
						Port:            2380,
						Username:        "admin",
						Password:        "secret",
						DialTimeout:     10 * time.Second,
						WorkloadsPrefix: "/custom/workloads/",
					},
					CRD: crd.Config{
						Namespace:    "crd-namespace",
						Kubeconfig:   "crd-kubeconfig",
						ResyncPeriod: 15 * time.Second,
					},
				},
			},
		},
		{
			Name:    "Default config",
			EnvVars: map[string]string{},
			ExpectedConfig: &Config{
				Host: DefaultServerHost,
				Port: DefaultServerPort,
				Store: store.Config{
					Type: DefaultStoreType,
					Etcd: etcd.Config{
						Host:            etcd.DefaultHost,
						Port:            etcd.DefaultPort,
						Username:        "",
						Password:        "",
						DialTimeout:     etcd.DefaultDialTimeout,
						WorkloadsPrefix: etcd.DefaultWorkloadsPrefix,
					},
					CRD: crd.Config{
						Namespace:    crd.DefaultNamespace,
						Kubeconfig:   "",
						ResyncPeriod: crd.DefaultResyncPeriod,
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
			require.NoError(t, err)
			assert.Equal(t, *test.ExpectedConfig, *config)
		})
	}
}
