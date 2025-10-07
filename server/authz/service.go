// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package authz

import (
	"context"
	"fmt"

	"github.com/agntcy/dir/server/authz/config"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc"
)

var logger = logging.Logger("authz")

// Service manages authorization policy enforcement.
// It expects authentication to be handled separately by the authn service,
// which will provide the SPIFFE ID in the context.
type Service struct {
	authorizer *Authorizer
}

// New creates a new authorization service.
func New(_ context.Context, cfg config.Config) (*Service, error) {
	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid authz config: %w", err)
	}

	// Create authorizer
	authorizer, err := NewAuthorizer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %w", err)
	}

	logger.Info("Authorization service initialized", "trust_domain", cfg.TrustDomain)

	return &Service{
		authorizer: authorizer,
	}, nil
}

// GetServerOptions returns gRPC server options for authorization.
func (s *Service) GetServerOptions() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(UnaryInterceptorFor(NewInterceptor(s.authorizer))),
		grpc.ChainStreamInterceptor(StreamInterceptorFor(NewInterceptor(s.authorizer))),
	}
}

// Stop closes any resources used by the authorization service.
func (s *Service) Stop() error {
	// No resources to clean up in the current implementation
	return nil
}
