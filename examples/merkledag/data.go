package main

import (
	"context"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"

	"github.com/agntcy/examples/merkledag/oci"
	v1 "github.com/agntcy/examples/merkledag/oci/v1"
)

func init() {
	// Register handlers for schema version 0.8.0
	oci.RegisterVersion("0.8.0", v1.Handlers())
	oci.RegisterVersion("v0.8.0", v1.Handlers())
}

// pushData converts an OASF record to an OCI manifest structure and pushes it
//
// Structure:
// - Config layer: Common metadata (name, version, schema_version, created_at, authors, description)
// - Each entity type (skills, domains, locators, modules) as independent layers
func pushData(ctx context.Context, repo oras.Target, record *oasfv1.Record) (ocispec.Descriptor, error) {
	return oci.Push(ctx, repo, record)
}

// pullData retrieves an OCI manifest and reconstructs the OASF record
func pullData(ctx context.Context, repo oras.Target, manifestDesc ocispec.Descriptor) (*oasfv1.Record, error) {
	return oci.Pull(ctx, repo, manifestDesc)
}
