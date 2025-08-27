// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

const (
	UnknownCID = "unknown"
)

func listCid(cmd *cobra.Command, client *client.Client, cid string) error {
	items, err := client.List(cmd.Context(), &routingv1.ListRequest{
		LegacyListRequest: &routingv1.LegacyListRequest{
			Ref: &corev1.RecordRef{
				Cid: cid,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list cid %s records: %w", cid, err)
	}

	// Print the results
	for item := range items {
		var cid string
		if ref := item.GetRef(); ref != nil {
			cid = ref.GetCid()
		} else {
			cid = UnknownCID
		}

		presenter.Printf(cmd,
			"Peer %s\n  CID: %s\n  Labels: %s\n",
			item.GetPeer().GetId(),
			cid,
			strings.Join(item.GetLabels(), ", "),
		)
	}

	return nil
}
