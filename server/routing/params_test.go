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
		RefreshInterval:         RefreshInterval,
		RecordTTL:               RecordTTL,
		DHTReprovideInterval:    DHTReprovideInterval,
		GossipReprovideInterval: GossipReprovideInterval,
		CleanupInterval:         CleanupInterval,
		MaxLabelAge:             MaxLabelAge,
		BucketSize:              BucketSize,
		Concurrency:             Concurrency,
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
				RefreshInterval:      time.Minute,
				RecordTTL:            12 * time.Hour,
				DHTReprovideInterval: 6 * time.Hour,
				GossipSub:            routingconfig.GossipSubConfig{ReprovideInterval: 30 * time.Minute},
				CleanupInterval:      time.Hour,
				MaxLabelAge:          12 * time.Hour,
				BucketSize:           20,
				Concurrency:          10,
			},
			want: dhtParams{
				RefreshInterval:         time.Minute,
				RecordTTL:               12 * time.Hour,
				DHTReprovideInterval:    6 * time.Hour,
				GossipReprovideInterval: 30 * time.Minute,
				CleanupInterval:         time.Hour,
				MaxLabelAge:             12 * time.Hour,
				BucketSize:              20,
				Concurrency:             10,
			},
		},
		{name: "negative value", cfg: routingconfig.Config{BucketSize: -1}, wantErr: true},
		{
			name:    "negative gossip interval",
			cfg:     routingconfig.Config{GossipSub: routingconfig.GossipSubConfig{ReprovideInterval: -time.Minute}},
			wantErr: true,
		},
		{name: "dht reprovide not below ttl", cfg: routingconfig.Config{DHTReprovideInterval: RecordTTL}, wantErr: true},
		{name: "label age not above dht reprovide", cfg: routingconfig.Config{MaxLabelAge: DHTReprovideInterval}, wantErr: true},
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
