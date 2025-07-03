// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"crypto/tls"
	"net/http"
	"time"
)

const (
	httpClientTimeout             = 30 * time.Second
	httpClientMaxIdleConns        = 30
	httpClientMaxIdleConnsPerHost = 10
	httpClientIdleConnTimeout     = 90 * time.Second
)

func CreateSecureHTTPClient(insecure bool) *http.Client {
	return &http.Client{
		Timeout: httpClientTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion:         tls.VersionTLS12,
				InsecureSkipVerify: insecure, // #nosec G402
			},
			MaxIdleConns:        httpClientMaxIdleConns,
			MaxIdleConnsPerHost: httpClientMaxIdleConnsPerHost,
			IdleConnTimeout:     httpClientIdleConnTimeout,
		},
	}
}
