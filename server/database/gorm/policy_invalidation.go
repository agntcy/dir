// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

// A stored verdict was reached on the metadata as it stood. When a writer
// changes that metadata the verdict stops describing the record, so the write
// supersedes it and the record is evaluated again.
//
// Writers call these to decide whether a write changed anything, because most
// writes change nothing: the reconciler re-reports the same scan result and
// re-verifies the same signature on every pass, and superseding on each of
// those would leave the verdict table never settled and the read gate hiding
// most of the catalogue most of the time.
//
// Two things bound what "changed" means here.
//
// Only what a policy can read counts. Each function compares the fields
// server/policy/meta.go projects and nothing else, so a column that exists for
// the reconciler's own bookkeeping — consecutive_failures, next_attempt_at,
// the stored certificate and public key — does not invalidate anything. These
// and the projection reach the row through the same types.*Object getters,
// which is what keeps the two from drifting apart.
//
// Timestamps are excluded, although the projection does expose them as
// scanned_at, verified_at and resolved_at. They record when this node last
// looked rather than what it concluded, and every re-report moves them, so
// counting them would make every write a change and defeat the whole
// mechanism. They also cannot be frozen instead: GetRecordsNeedingScan bounds
// on scan_reports.updated_at to schedule rescans, so a row whose timestamp
// stopped moving would look permanently expired. The consequence is worth
// stating plainly — a policy whose answer depends on the clock, such as one
// requiring a scan newer than a day, cannot be kept correct by invalidation at
// all, because its answer drifts while nothing writes.
//
// The comparison is per row, against the row the write replaces. An aggregate
// cannot move unless one of its rows appears, changes or goes away, so this
// never misses a change. It can fire when a row moves and the aggregate does
// not: max_severity is a maximum across scanner rows, so a lower row changing
// supersedes verdicts that will be recomputed to the same answer. That costs
// an evaluation, which is the cheap direction to be wrong in.
//
// Writes that add or remove a row need no comparison, because appearing and
// disappearing are changes in themselves. Only a write that replaces a row
// has to ask.

// scanFactsChanged reports whether replacing before with after changes what
// scanFacts projects. A caller with no row to replace has a new fact rather
// than a changed one and does not need to ask.
func scanFactsChanged(before, after *ScanReport) bool {
	return before.Status != after.Status ||
		before.MaxSeverity != after.MaxSeverity ||
		before.IsSafe != after.IsSafe ||
		before.Source != after.Source
}

// signatureVerificationFactsChanged reports whether replacing before with
// after changes what signatureFacts projects.
//
// The signer fields matter because the projection reports them, not because
// they identify the row; signer_key does that and the write is keyed on it.
func signatureVerificationFactsChanged(before, after *SignatureVerification) bool {
	return before.Status != after.Status ||
		before.SignerSubject != after.SignerSubject ||
		before.SignerIssuer != after.SignerIssuer
}

// identityClaimFactsChanged reports whether replacing before with after
// changes what claimFacts projects.
//
// The subject counts even though the record declares one of its own: the
// projection prefers the stored subject when there is one, so a claim that
// resolves to a different subject changes what a policy reads.
func identityClaimFactsChanged(before, after *IdentityClaim) bool {
	return before.Status != after.Status ||
		before.Subject != after.Subject
}
