// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import "testing"

// ResetDefaultValidatorForTest clears the package-level default validator
// so unit tests can exercise the "uninitialized" path of the deprecated
// (*Record).Validate(ctx) wrapper deterministically.
//
// It accepts a testing.TB to make accidental production calls a compile
// error. It is intentionally exported (rather than living in an _test.go
// file) so external packages such as server/config tests can also reset
// the default state when they need to.
//
// Deprecated: this helper exists only to support the back-compat path
// that itself is deprecated. It will be removed alongside Validate(ctx)
// and InitializeValidator in a future major version. See
// https://github.com/agntcy/dir/issues/856.
func ResetDefaultValidatorForTest(tb testing.TB) {
	tb.Helper()

	defaultValidMu.Lock()
	defaultValidator = nil
	defaultValidMu.Unlock()
}
