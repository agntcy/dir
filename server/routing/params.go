// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"time"

	routingconfig "github.com/agntcy/dir/server/routing/config"
)

// dhtParams holds the DHT and routing timing settings after applying defaults.
type dhtParams struct {
	RefreshInterval         time.Duration
	RecordTTL               time.Duration
	DHTReprovideInterval    time.Duration
	GossipReprovideInterval time.Duration
	CleanupInterval         time.Duration
	MaxLabelAge             time.Duration
	BucketSize              int
	Concurrency             int
}

// resolveDHTParams fills unset (zero) config values with the package defaults
// and rejects invalid combinations.
func resolveDHTParams(cfg routingconfig.Config) (dhtParams, error) {
	if cfg.RefreshInterval < 0 || cfg.RecordTTL < 0 || cfg.DHTReprovideInterval < 0 || cfg.GossipSub.ReprovideInterval < 0 ||
		cfg.CleanupInterval < 0 || cfg.MaxLabelAge < 0 || cfg.BucketSize < 0 || cfg.Concurrency < 0 {
		return dhtParams{}, errors.New("routing intervals, bucket_size and concurrency must not be negative")
	}

	p := dhtParams{
		RefreshInterval:         orDefault(cfg.RefreshInterval, RefreshInterval),
		RecordTTL:               orDefault(cfg.RecordTTL, RecordTTL),
		DHTReprovideInterval:    orDefault(cfg.DHTReprovideInterval, DHTReprovideInterval),
		GossipReprovideInterval: orDefault(cfg.GossipSub.ReprovideInterval, GossipReprovideInterval),
		CleanupInterval:         orDefault(cfg.CleanupInterval, CleanupInterval),
		MaxLabelAge:             orDefault(cfg.MaxLabelAge, MaxLabelAge),
		BucketSize:              orDefault(cfg.BucketSize, BucketSize),
		Concurrency:             orDefault(cfg.Concurrency, Concurrency),
	}

	// Records must be reprovided before they expire on remote peers.
	if p.DHTReprovideInterval >= p.RecordTTL {
		return dhtParams{}, fmt.Errorf("dht_reprovide_interval (%s) must be lower than record_ttl (%s)",
			p.DHTReprovideInterval, p.RecordTTL)
	}

	// Remote labels are also refreshed by DHT provider notifications, so they must
	// not go stale before the next DHT reprovide, even with GossipSub disabled.
	if p.MaxLabelAge <= p.DHTReprovideInterval {
		return dhtParams{}, fmt.Errorf("max_label_age (%s) must be greater than dht_reprovide_interval (%s)",
			p.MaxLabelAge, p.DHTReprovideInterval)
	}

	return p, nil
}

func orDefault[T int | time.Duration](v, def T) T {
	if v > 0 {
		return v
	}

	return def
}
