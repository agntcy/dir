// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"errors"
	"net/http"

	"oras.land/oras-go/v2/errdef"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

// IsNotFound reports whether err says the registry does not hold what was
// asked for. oras-go reports a missing manifest, blob or tag target as
// errdef.ErrNotFound, and a missing repository as an error response with
// status 404; both are recognized by type. The error text is never matched:
// it carries the registry's address, which may itself contain "404".
func IsNotFound(err error) bool {
	if errors.Is(err, errdef.ErrNotFound) {
		return true
	}

	var resp *errcode.ErrorResponse

	return errors.As(err, &resp) && resp.StatusCode == http.StatusNotFound
}

// isDeleteUnsupported reports whether err says the registry does not allow
// deleting through the OCI API: a 405 response, an UNSUPPORTED error code, or
// oras-go's errdef.ErrUnsupported. As with IsNotFound, the error text is not
// matched, since an address can contain "405" as well.
func isDeleteUnsupported(err error) bool {
	if errors.Is(err, errdef.ErrUnsupported) {
		return true
	}

	var resp *errcode.ErrorResponse

	if !errors.As(err, &resp) {
		return false
	}

	if resp.StatusCode == http.StatusMethodNotAllowed {
		return true
	}

	for _, e := range resp.Errors {
		if e.Code == errcode.ErrorCodeUnsupported {
			return true
		}
	}

	return false
}
