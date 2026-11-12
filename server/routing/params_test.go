// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"testing"
	"time"

	routingconfig "github.com/agntcy/dir/server/routing/config"
)

func TestResolveDHTParams(t *testing.T) {
	defaults := dhtParams{
		RefreshInterval:   RefreshInterval,
		RecordTTL:         RecordTTL,
		RepublishInterval: RepublishInterval,
		CleanupInterval:   CleanupInterval,
		MaxLabelAge:       MaxLabelAge,
		BucketSize:        BucketSize,
		Concurrency:       Concurrency,
	}

	tests := []struct {
		name    string
		cfg     routingconfig.Config
		want    dhtParams
		wantErr bool
	}{
		{name: "empty config uses defaults", want: defaults},
		{
			name: "overrides are applied",
			cfg: routingconfig.Config{
				RefreshInterval:   time.Minute,
				RecordTTL:         12 * time.Hour,
				RepublishInterval: 6 * time.Hour,
				CleanupInterval:   time.Hour,
				MaxLabelAge:       12 * time.Hour,
				BucketSize:        20,
				Concurrency:       10,
			},
			want: dhtParams{
				RefreshInterval:   time.Minute,
				RecordTTL:         12 * time.Hour,
				RepublishInterval: 6 * time.Hour,
				CleanupInterval:   time.Hour,
				MaxLabelAge:       12 * time.Hour,
				BucketSize:        20,
				Concurrency:       10,
			},
		},
		{name: "negative value", cfg: routingconfig.Config{BucketSize: -1}, wantErr: true},
		{name: "republish not below ttl", cfg: routingconfig.Config{RepublishInterval: RecordTTL}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveDHTParams(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
