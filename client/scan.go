// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
)

// PushScanReport stores a ScanReport as a referrer attached to the given record.
// Failures are returned so the caller can decide whether to treat them as fatal.
func (c *Client) PushScanReport(ctx context.Context, recordRef *corev1.RecordRef, report *securityv1.ScanReport) error {
	referrer, err := report.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("marshal scan report: %w", err)
	}

	_, err = c.PushReferrer(ctx, &storev1.PushReferrerRequest{
		RecordRef:   recordRef,
		Type:        referrer.GetType(),
		Annotations: referrer.GetAnnotations(),
		Data:        referrer.GetData(),
	})
	if err != nil {
		return fmt.Errorf("push scan report referrer: %w", err)
	}

	return nil
}
