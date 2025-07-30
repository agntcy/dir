// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"encoding/json"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "pull",
	Short: "Pull agent model from Directory server",
	Long: `This command pulls the agent data model from Directory API. The data can be validated against its hash, as
the returned object is content-addressable.

Usage examples:

1. Pull by digest and output

	dirctl pull <digest>

2. Pull by digest and include signature:

	dirctl pull <digest> --include-signature

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("digest is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, digest string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	var record *corev1.Record

	var signature *signv1.Signature

	var err error

	// Fetch record from store
	if opts.IncludeSignature {
		resp, err := c.PullWithOptions(cmd.Context(), &corev1.RecordRef{
			Cid: digest, // Use digest as CID directly
		}, true)
		if err != nil {
			return fmt.Errorf("failed to pull data: %w", err)
		}

		record = resp.GetRecord()
		signature = resp.GetSignature()
	} else {
		record, err = c.Pull(cmd.Context(), &corev1.RecordRef{
			Cid: digest, // Use digest as CID directly
		})
		if err != nil {
			return fmt.Errorf("failed to pull data: %w", err)
		}
	}

	// Extract the OASF object from the Record based on version
	var oasfData interface{}

	var rawData []byte

	switch data := record.GetData().(type) {
	case *corev1.Record_V1:
		oasfData = data.V1
		rawData, err = json.Marshal(data.V1)
	case *corev1.Record_V2:
		oasfData = data.V2
		rawData, err = json.Marshal(data.V2)
	case *corev1.Record_V3:
		oasfData = data.V3
		rawData, err = json.Marshal(data.V3)
	default:
		return errors.New("unsupported record type")
	}

	if err != nil {
		return fmt.Errorf("failed to marshal OASF object: %w", err)
	}

	// If raw format flag is set, print and exit
	if opts.FormatRaw {
		presenter.Print(cmd, string(rawData))

		return nil
	}

	// Pretty-print the OASF object
	output, err := json.MarshalIndent(oasfData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OASF object to JSON: %w", err)
	}

	presenter.Print(cmd, string(output))

	if opts.IncludeSignature {
		signatureJSON, err := json.MarshalIndent(signature, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal signature to JSON: %w", err)
		}

		presenter.Print(cmd, string(signatureJSON))
	}

	return nil
}
