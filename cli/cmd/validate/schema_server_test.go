// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// The unit tests in this package used to point --url at the live schema service
// at https://schema.oasf.outshift.com. That makes the unit job depend on a
// remote host being reachable and fast: it goes red on a TLS handshake timeout
// with no code change, which is what happened on CI (`TestValidateCommand_NoFileArgs`,
// ~19s, "net/http: TLS handshake timeout"). A unit suite that fails on someone
// else's network is a suite people learn to re-run rather than read.
//
// So they now point at a local httptest server by default. The responses it
// serves are RECORDED from the real service, not invented -- see the
// testdata/schema_response_*.json files and the note below -- so the fake
// cannot drift into asserting a protocol that does not exist.
//
// Set OASF_SCHEMA_URL to run against a real service instead:
//
//	OASF_SCHEMA_URL=https://schema.oasf.outshift.com go test ./cli/cmd/validate/...
//
// That is the path to use when checking for contract drift, and the reason this
// is an env override rather than a hard-coded fake: the live check stays
// available, it just stops being the default for every unit run.

// Recorded from POST /api/0.8.0/validate/object/record?missing_recommended=true
// on 2026-07-31, one per request fixture. Captured verbatim; if the service's
// contract changes, an OASF_SCHEMA_URL run is what surfaces it.
//
//go:embed testdata/schema_response_valid.json
var schemaResponseValid []byte

//go:embed testdata/schema_response_invalid.json
var schemaResponseInvalid []byte

//go:embed testdata/schema_response_valid_with_warnings.json
var schemaResponseValidWithWarnings []byte

// schemaURL returns the base URL the --url flag should be given.
//
// OASF_SCHEMA_URL wins when set. Otherwise a per-test httptest server is
// started and torn down with the test.
func schemaURL(t *testing.T) string {
	t.Helper()

	if live := os.Getenv("OASF_SCHEMA_URL"); live != "" {
		return live
	}

	srv := httptest.NewServer(http.HandlerFunc(serveRecordedValidation))
	t.Cleanup(srv.Close)

	return srv.URL
}

// serveRecordedValidation replays the recorded response for whichever request
// fixture it was handed.
//
// Dispatch is on the semantic content of the record rather than a byte compare,
// so reformatting a fixture does not silently turn every response into the
// valid one -- a fake that answers "valid" to an unrecognised body would make
// these tests pass for the wrong reason.
func serveRecordedValidation(w http.ResponseWriter, r *http.Request) {
	body := new(bytes.Buffer)
	if _, err := body.ReadFrom(r.Body); err != nil {
		http.Error(w, "read body", http.StatusBadRequest)

		return
	}

	resp, ok := recordedResponseFor(body.Bytes())
	if !ok {
		// Loud on purpose. A new fixture with no recorded response must fail
		// visibly rather than inherit someone else's answer.
		http.Error(w, "no recorded response for this record fixture", http.StatusNotImplemented)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

func recordedResponseFor(body []byte) ([]byte, bool) {
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		return nil, false
	}

	for _, fixture := range []struct {
		record   []byte
		response []byte
	}{
		{testRecordValid, schemaResponseValid},
		{testRecordInvalid, schemaResponseInvalid},
		{testRecordValidWithWarnings, schemaResponseValidWithWarnings},
	} {
		var want map[string]any
		if err := json.Unmarshal(fixture.record, &want); err != nil {
			continue
		}

		if sameRecord(got, want) {
			return fixture.response, true
		}
	}

	return nil, false
}

// sameRecord compares the fields that distinguish this package's three
// fixtures. Full deep equality would be stricter than needed and would break
// the moment the CLI adds or normalises a field on the way out.
func sameRecord(got, want map[string]any) bool {
	for _, key := range []string{"name", "version", "previous_record_cid"} {
		if got[key] != want[key] {
			return false
		}
	}

	return true
}
