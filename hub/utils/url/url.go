// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package url

import (
	"errors"
	"net/url"
)

func ValidateSecureURL(u string) error {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return errors.New("invalid URL format")
	}

	if parsedURL.Scheme != "https" {
		return errors.New("URL scheme must be HTTPS")
	}

	return nil
}
