// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package translate provides the CLI command for translating MCP/A2A configurations to OASF objects using the Agent Hub.
package translate

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/agntcy/dir/hub/api/v1alpha1"
	"github.com/agntcy/dir/hub/auth"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// translateOptions holds the configuration for the translate command
type translateOptions struct {
	configType string
	inputFile  string
	outputFile string
	stdin      bool
}

// addAuthToContext adds the authorization header to the context if an access token is available.
func addAuthToContext(ctx context.Context, session *sessionstore.HubSession) context.Context {
	if session != nil && session.Tokens != nil && session.CurrentTenant != "" {
		if t, ok := session.Tokens[session.CurrentTenant]; ok && t != nil && t.AccessToken != "" {
			return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+t.AccessToken))
		}
	}
	return ctx
}

// NewCommand creates the "translate" command for the Agent Hub CLI.
// It translates MCP/A2A configurations to OASF objects using the translation service.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *hubOptions.HubOptions) *cobra.Command {
	opts := &translateOptions{}

	cmd := &cobra.Command{
		Use:   "translate [flags] {<config_file> | --stdin}",
		Short: "Translate MCP/A2A configurations to OASF objects",
		Long: `Translate MCP (Model Context Protocol) or A2A (Agent-to-Agent) configurations to OASF (Open Agent Standard Format) objects.

The command accepts configuration data either from a file or stdin and uses the Agent Hub's translation service to convert it.

Parameters:
  <config_file>   Path to the MCP or A2A configuration file (optional if --stdin is used)
  --stdin         Read configuration from standard input (optional)
  --type          Type of configuration (mcp or a2a) - required
  --output        Output file path (optional, defaults to stdout)

Examples:
  # Translate MCP configuration from file
  dirctl hub translate --type mcp config.json

  # Translate A2A configuration from stdin
  cat config.json | dirctl hub translate --type a2a --stdin

  # Translate and save to file
  dirctl hub translate --type mcp --output agent.json config.json`,

		RunE: func(cmd *cobra.Command, args []string) error {
			// If no arguments and no required flags, show help
			if len(args) == 0 && opts.configType == "" && !opts.stdin {
				return cmd.Help()
			}

			// Validate arguments and flags
			if err := validateTranslateArgs(opts, args); err != nil {
				// Show help on validation error
				cmd.Help()
				return err
			}

			// Retrieve session from context
			ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
			currentSession, ok := ctxSession.(*sessionstore.HubSession)
			if !ok || !auth.HasLoginCreds(currentSession) {
				return errors.New("authentication required: please run 'dirctl hub login' first")
			}

			// Read input data
			inputData, err := readInputData(opts, args)
			if err != nil {
				return fmt.Errorf("failed to read input data: %w", err)
			}

			// Translate the configuration
			result, err := translateConfiguration(cmd.Context(), currentSession, opts.configType, inputData)
			if err != nil {
				return fmt.Errorf("translation failed: %w", err)
			}

			// Write output
			if err := writeOutput(opts, result); err != nil {
				return fmt.Errorf("failed to write output: %w", err)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVar(&opts.configType, "type", "", "Configuration type (mcp or a2a) - required")
	cmd.Flags().StringVarP(&opts.outputFile, "output", "o", "", "Output file path (optional, defaults to stdout)")
	cmd.Flags().BoolVar(&opts.stdin, "stdin", false, "Read configuration from standard input")

	return cmd
}

// validateTranslateArgs validates the command arguments and flags
func validateTranslateArgs(opts *translateOptions, args []string) error {
	// Validate config type
	if opts.configType != "mcp" && opts.configType != "a2a" {
		return errors.New("--type must be either 'mcp' or 'a2a'")
	}

	// Validate input source
	if opts.stdin {
		if len(args) > 0 {
			return errors.New("cannot specify both input file and --stdin")
		}
	} else {
		if len(args) != 1 {
			return errors.New("exactly one configuration file must be specified when not using --stdin")
		}
		opts.inputFile = args[0]
	}

	return nil
}

// readInputData reads the configuration data from file or stdin
func readInputData(opts *translateOptions, args []string) ([]byte, error) {
	var reader io.Reader

	if opts.stdin {
		reader = os.Stdin
	} else {
		file, err := os.Open(opts.inputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open file '%s': %w", opts.inputFile, err)
		}
		defer file.Close()
		reader = file
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	if len(data) == 0 {
		return nil, errors.New("input data is empty")
	}

	return data, nil
}

// translateConfiguration calls the translation service to convert the configuration
func translateConfiguration(ctx context.Context, session *sessionstore.HubSession, configType string, inputData []byte) ([]byte, error) {
	// Get the server address from the session
	if session.AuthConfig == nil {
		return nil, errors.New("missing authentication configuration")
	}

	serverAddr := session.AuthConfig.HubBackendAddress

	// Create gRPC connection
	conn, err := grpc.NewClient(
		serverAddr,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc connection: %w", err)
	}
	defer conn.Close()

	// Create translation client
	client := v1alpha1.NewTranslationServiceClient(conn)

	// Add authentication to context
	authCtx := addAuthToContext(ctx, session)

	// Encode input data as base64
	encodedData := base64.StdEncoding.EncodeToString(inputData)

	// Create translation request
	request := &v1alpha1.TranslateRequest{
		Type: configType,
		Data: encodedData,
	}

	// Call the translation service
	response, err := client.Translate(authCtx, request)
	if err != nil {
		return nil, fmt.Errorf("translation service call failed: %w", err)
	}

	// Marshal the response to JSON
	result, err := json.MarshalIndent(response.Record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	return result, nil
}

// writeOutput writes the translated result to the specified output
func writeOutput(opts *translateOptions, result []byte) error {
	if opts.outputFile == "" {
		// Write to stdout
		_, err := os.Stdout.Write(result)
		if err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
		fmt.Println() // Add newline
	} else {
		// Write to file
		err := os.WriteFile(opts.outputFile, result, 0644)
		if err != nil {
			return fmt.Errorf("failed to write to file '%s': %w", opts.outputFile, err)
		}
		fmt.Printf("Translation completed successfully. Output written to: %s\n", opts.outputFile)
	}

	return nil
}
