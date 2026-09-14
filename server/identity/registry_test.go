// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"context"
	"errors"
	"testing"

	ansconfig "github.com/agntcy/dir/server/identity/ans/config"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const ansSubject = "ans://v1.0.0.agent.example.com"

// fakeResolver answers Verify with fixed values and records the subject it saw.
type fakeResolver struct {
	scheme  string
	ok      bool
	err     error
	subject string
}

func (f *fakeResolver) Scheme() string { return f.scheme }

func (f *fakeResolver) Verify(_ context.Context, subject string, _, _ []byte) (bool, error) {
	f.subject = subject

	return f.ok, f.err
}

func TestRegistry_Verify(t *testing.T) {
	errUpstream := errors.New("dns: lookup failed")

	tests := []struct {
		name      string
		resolvers []*fakeResolver
		subject   string
		wantOK    bool
		// wantScheme is the scheme of the resolver that must have seen the subject.
		wantScheme string
		wantErr    string
		wantErrIs  error
	}{
		{
			name:       "https subject reaches the https resolver",
			resolvers:  []*fakeResolver{{scheme: "https", ok: true}, {scheme: "dns"}},
			subject:    "https://example.com",
			wantOK:     true,
			wantScheme: "https",
		},
		{
			name:       "bare host reaches the dns resolver",
			resolvers:  []*fakeResolver{{scheme: "https"}, {scheme: "dns", ok: true}},
			subject:    "example.com",
			wantOK:     true,
			wantScheme: "dns",
		},
		{
			name:       "ans subject reaches the ans resolver",
			resolvers:  []*fakeResolver{{scheme: "dns"}, {scheme: "ans", ok: true}},
			subject:    ansSubject,
			wantOK:     true,
			wantScheme: "ans",
		},
		{
			name:       "no matching key is a verdict, not an error",
			resolvers:  []*fakeResolver{{scheme: "dns"}},
			subject:    "example.com",
			wantScheme: "dns",
		},
		{
			name:       "a resolver error keeps its text",
			resolvers:  []*fakeResolver{{scheme: "dns", err: errUpstream}},
			subject:    "example.com",
			wantScheme: "dns",
			wantErr:    errUpstream.Error(),
			wantErrIs:  errUpstream,
		},
		{
			name:      "a scheme without a resolver",
			resolvers: []*fakeResolver{{scheme: "dns"}},
			subject:   ansSubject,
			wantErr:   `no resolver registered for subject scheme "ans"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvers := make([]Resolver, 0, len(tt.resolvers))
			for _, r := range tt.resolvers {
				resolvers = append(resolvers, r)
			}

			ok, err := NewRegistry(resolvers...).Verify(t.Context(), tt.subject, []byte("sig"), []byte("payload"))

			for _, r := range tt.resolvers {
				if r.scheme == tt.wantScheme {
					assert.Equal(t, tt.subject, r.subject, "the %s resolver must see the subject", r.scheme)
				} else {
					assert.Empty(t, r.subject, "the %s resolver must not be called", r.scheme)
				}
			}

			if tt.wantErr != "" {
				assert.False(t, ok)
				require.EqualError(t, err, tt.wantErr)

				if tt.wantErrIs != nil {
					require.ErrorIs(t, err, tt.wantErrIs)
				}

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestNewDefaultRegistry(t *testing.T) {
	enabled := ansconfig.Config{
		Enabled:               true,
		TrustedLogHosts:       []string{"log.example.com"},
		AllowUnpinnedRootKeys: true,
	}

	tests := []struct {
		name    string
		cfg     ansconfig.Config
		wantErr string
		// wantAnsErr is what an ans:// claim with an unusable signature gets:
		// unrouted without the resolver, refused by its certificate stage with it.
		wantAnsErr string
	}{
		{
			name:       "ans off leaves ans:// subjects unrouted",
			wantAnsErr: `no resolver registered for subject scheme "ans"`,
		},
		{
			name:       "ans on routes ans:// subjects to the ans resolver",
			cfg:        enabled,
			wantAnsErr: "ans certificate:",
		},
		{
			name:    "an ans block that fails validation is an error",
			cfg:     ansconfig.Config{Enabled: true, AllowUnpinnedRootKeys: true},
			wantErr: "trusted_log_hosts",
		},
		{
			name:    "a malformed pinned root key is an error",
			cfg:     ansconfig.Config{Enabled: true, TrustedLogHosts: []string{"log.example.com"}, RootKeys: []string{"not-a-key-line"}},
			wantErr: "root_keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, err := NewDefaultRegistry(tt.cfg, safefetch.New())
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, registry)

				return
			}

			require.NoError(t, err)

			for _, scheme := range []string{"https", "dns", "did"} {
				assert.Contains(t, registry.resolvers, scheme)
			}

			_, hasAns := registry.resolvers["ans"]
			assert.Equal(t, tt.cfg.Enabled, hasAns)

			ok, err := registry.Verify(t.Context(), ansSubject, []byte("not a jws"), []byte("payload"))
			assert.False(t, ok)
			require.ErrorContains(t, err, tt.wantAnsErr)
		})
	}
}
