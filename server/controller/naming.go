// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package controller

import (
	"context"
	"errors"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	namingv1 "github.com/agntcy/dir/api/naming/v1"
	"github.com/agntcy/dir/server/database/sqlite"
	"github.com/agntcy/dir/server/naming"
	reverificationconfig "github.com/agntcy/dir/server/reverification/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/agntcy/dir/utils/logging"
	"golang.org/x/mod/semver"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var namingLogger = logging.Logger("controller/naming")

type namingCtrl struct {
	namingv1.UnimplementedNamingServiceServer
	store    types.StoreAPI
	db       types.DatabaseAPI
	provider *naming.Provider
	ttl      time.Duration
}

// NamingControllerOption configures a naming controller.
type NamingControllerOption func(*namingCtrl)

// WithVerificationTTL sets the verification TTL.
func WithVerificationTTL(ttl time.Duration) NamingControllerOption {
	return func(n *namingCtrl) {
		n.ttl = ttl
	}
}

// NewNamingController creates a new naming service controller.
func NewNamingController(store types.StoreAPI, db types.DatabaseAPI, provider *naming.Provider, opts ...NamingControllerOption) namingv1.NamingServiceServer {
	ctrl := &namingCtrl{
		store:    store,
		db:       db,
		provider: provider,
		ttl:      reverificationconfig.DefaultTTL,
	}

	for _, opt := range opts {
		opt(ctrl)
	}

	return ctrl
}

// GetVerificationInfo retrieves the verification info for a record.
func (n *namingCtrl) GetVerificationInfo(ctx context.Context, req *namingv1.GetVerificationInfoRequest) (*namingv1.GetVerificationInfoResponse, error) {
	namingLogger.Debug("GetVerificationInfo request received", "cid", req.GetCid())

	if req.GetCid() == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required")
	}

	// Query database for the latest verification attempt
	latest, err := n.db.GetVerificationByCID(req.GetCid())
	if err != nil {
		if errors.Is(err, sqlite.ErrVerificationNotFound) {
			errMsg := "no verification found"

			return &namingv1.GetVerificationInfoResponse{
				Verified:     false,
				ErrorMessage: &errMsg,
			}, nil
		}

		return nil, status.Errorf(codes.Internal, "failed to get verification: %v", err)
	}

	// Check if the latest verification is valid (verified and not expired)
	if !n.isVerificationValid(latest) {
		errMsg := latest.GetError()
		if errMsg == "" {
			errMsg = "verification invalid or expired"
		}

		return &namingv1.GetVerificationInfoResponse{
			Verified:     false,
			ErrorMessage: &errMsg,
		}, nil
	}

	// Return valid verification from database
	namingLogger.Debug("Returning verification from database", "cid", req.GetCid())

	return &namingv1.GetVerificationInfoResponse{
		Verified: true,
		Verification: namingv1.NewDomainVerification(&namingv1.DomainVerification{
			Domain:     n.getDomainFromRecord(ctx, req.GetCid()),
			Method:     latest.GetMethod(),
			KeyId:      latest.GetKeyID(),
			VerifiedAt: timestamppb.New(latest.GetUpdatedAt()),
		}),
	}, nil
}

// isVerificationValid checks if a verification is valid (verified status and not expired).
func (n *namingCtrl) isVerificationValid(v types.NameVerificationObject) bool {
	if v.GetStatus() != sqlite.VerificationStatusVerified {
		return false
	}

	// Check if verification has expired: updated_at + ttl < now
	return time.Now().Before(v.GetUpdatedAt().Add(n.ttl))
}

// getDomainFromRecord extracts the domain from a record's name.
func (n *namingCtrl) getDomainFromRecord(ctx context.Context, cid string) string {
	record, err := n.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return ""
	}

	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		return ""
	}

	parsed := naming.ParseName(recordData.GetName())
	if parsed == nil {
		return ""
	}

	return parsed.Domain
}

// Resolve resolves a record reference (name with optional version) to a single CID.
// If version is specified, returns the exact match.
// If no version is specified, returns the latest version by semver.
// Names without protocol prefix are searched with both http:// and https://.
func (n *namingCtrl) Resolve(ctx context.Context, req *namingv1.ResolveRequest) (*namingv1.ResolveResponse, error) {
	namingLogger.Debug("Resolve request received", "name", req.GetName(), "version", req.GetVersion())

	if req.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Build filter options: filter by name (with protocol variations if not specified)
	filterOptions := []types.FilterOption{
		types.WithNames(expandNameWithProtocols(req.GetName())...),
	}

	// Add version filter if specified
	if req.GetVersion() != "" {
		filterOptions = append(filterOptions, types.WithVersions(req.GetVersion()))
	}

	// Get matching records
	records, err := n.db.GetRecords(filterOptions...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to search records: %v", err)
	}

	if len(records) == 0 {
		if req.GetVersion() != "" {
			return nil, status.Errorf(codes.NotFound, "no record found with name %q and version %q", req.GetName(), req.GetVersion())
		}

		return nil, status.Errorf(codes.NotFound, "no record found with name %q", req.GetName())
	}

	// If version was specified and we have multiple matches, it's ambiguous
	if req.GetVersion() != "" && len(records) > 1 {
		return nil, status.Errorf(codes.FailedPrecondition,
			"multiple records found with name %q and version %q, pull by CID directly",
			req.GetName(), req.GetVersion())
	}

	// If only one match, return it
	if len(records) == 1 {
		return &namingv1.ResolveResponse{
			Cid: records[0].GetCid(),
		}, nil
	}

	// Multiple matches without version specified - find the latest by semver
	return n.findLatestBySemver(records)
}

// findLatestBySemver finds the record with the highest semver version from the given records.
func (n *namingCtrl) findLatestBySemver(records []types.Record) (*namingv1.ResolveResponse, error) {
	if len(records) == 0 {
		return nil, status.Error(codes.Internal, "no records provided")
	}

	// Find the record with the highest semver version
	latest := records[0]

	for _, r := range records[1:] {
		latestData, _ := latest.GetRecordData()
		rData, _ := r.GetRecordData()

		if compareSemver(rData.GetVersion(), latestData.GetVersion()) > 0 {
			latest = r
		}
	}

	return &namingv1.ResolveResponse{
		Cid: latest.GetCid(),
	}, nil
}

// compareSemver compares two version strings using semver.
// Returns >0 if a > b, <0 if a < b, 0 if equal.
// Handles versions with or without 'v' prefix.
// Falls back to string comparison if not valid semver.
func compareSemver(a, b string) int {
	va := ensureVPrefix(a)
	vb := ensureVPrefix(b)

	if semver.IsValid(va) && semver.IsValid(vb) {
		return semver.Compare(va, vb)
	}

	// Fall back to string comparison
	if a > b {
		return 1
	} else if a < b {
		return -1
	}

	return 0
}

// ensureVPrefix adds 'v' prefix if not present (semver package requires it).
func ensureVPrefix(version string) string {
	if version == "" {
		return ""
	}

	if version[0] != 'v' {
		return "v" + version
	}

	return version
}

// expandNameWithProtocols returns name variations to search for.
// If the name already has a protocol prefix, returns it as-is.
// Otherwise, returns exact match plus http:// and https:// variations.
// This allows finding both:
// - Records with protocol prefixes (verifiable names like "https://cisco.com/agent")
// - Records without protocol prefixes (non-verifiable names like "my-org/agent").
func expandNameWithProtocols(name string) []string {
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") {
		return []string{name}
	}

	return []string{
		name, // exact match for non-verifiable names
		"http://" + name,
		"https://" + name,
	}
}
