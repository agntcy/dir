// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinal(t *testing.T) {
	dnsErr := &net.DNSError{Err: "server misbehaving", Name: "_ans-badge.acme.com"}

	tests := []struct {
		name     string
		err      error
		wantText string
		wantIs   []error
		wantDNS  bool
	}{
		{name: "nil stays nil"},
		{
			name:     "a plain error",
			err:      errors.New("ans badge: no record"),
			wantText: "ans badge: no record",
			wantIs:   []error{ErrFinal},
		},
		{
			name:     "a wrapped cause stays readable",
			err:      fmt.Errorf("ans badge: lookup: %w", dnsErr),
			wantText: "ans badge: lookup: lookup _ans-badge.acme.com: server misbehaving",
			wantIs:   []error{ErrFinal},
			wantDNS:  true,
		},
		{
			name:     "a context error stays readable",
			err:      fmt.Errorf("ans log: fetch: %w", context.DeadlineExceeded),
			wantText: "ans log: fetch: context deadline exceeded",
			wantIs:   []error{ErrFinal, context.DeadlineExceeded},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Final(tt.err)
			if tt.err == nil {
				require.NoError(t, got)

				return
			}

			require.EqualError(t, got, tt.wantText)

			for _, want := range tt.wantIs {
				require.ErrorIs(t, got, want)
			}

			var gotDNS *net.DNSError

			assert.Equal(t, tt.wantDNS, errors.As(got, &gotDNS))
		})
	}

	// An error that is not marked is not final.
	require.NotErrorIs(t, errors.New("x"), ErrFinal)
	require.NotErrorIs(t, fmt.Errorf("fetch: %w", dnsErr), ErrFinal)
}
