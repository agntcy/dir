// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import "testing"

// TestFreeCString is a smoke test that FreeCString does not panic or crash
// when releasing a freshly allocated C string. There is no observable return
// value to assert on; the test simply exercises the call.
func TestFreeCString(t *testing.T) {
	bridgeFreeCStringSmoke("hello from FreeCString smoke test")
}
