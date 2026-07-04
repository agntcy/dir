// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides the canonical configuration schema for the
// dir project. Both the standalone apiserver, the standalone reconciler,
// and the embedded daemon read from a single Config struct, ensuring that
// shared infrastructure (OCI registry, database) is declared once and never
// duplicated across service boundaries.
//
// Usage:
//
//	cfg, err := config.LoadConfig()           // DIRECTORY_* env + /etc/agntcy/dir/dir.config.yml
//	cfg, err := config.LoadConfig(
//	    config.WithFile("/path/to/config.yml"), // explicit file
//	    config.WithEnvPrefix("MY_PREFIX"),       // override env prefix
//	)
package config
