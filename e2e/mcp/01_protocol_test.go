// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// MCPRequest represents a JSON-RPC 2.0 request.
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// MCPResponse represents a JSON-RPC 2.0 response.
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC 2.0 error.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPClient manages the MCP server process and communication.
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr *bufio.Scanner
}

// NewMCPClient starts an MCP server and returns a client to communicate with it.
// The path parameter should be the directory containing the MCP server code.
func NewMCPClient(mcpDir string) (*MCPClient, error) {
	cmd := exec.CommandContext(context.Background(), "go", "run", ".")
	cmd.Dir = mcpDir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server: %w", err)
	}

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Create scanner with larger buffer for large responses (e.g., schema resources)
	stdoutScanner := bufio.NewScanner(stdout)

	const maxTokenSize = 10 * 1024 * 1024 // 10MB

	buf := make([]byte, maxTokenSize)
	stdoutScanner.Buffer(buf, maxTokenSize)

	return &MCPClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdoutScanner,
		stderr: bufio.NewScanner(stderr),
	}, nil
}

// SendRequest sends a JSON-RPC request and returns the response.
func (c *MCPClient) SendRequest(req MCPRequest) (*MCPResponse, error) {
	// Marshal request
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request with newline
	if _, err := c.stdin.Write(append(reqBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
	if !c.stdout.Scan() {
		if err := c.stdout.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}

		return nil, errors.New("no response received")
	}

	// Parse response
	var resp MCPResponse
	if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// Close stops the MCP server and cleans up.
func (c *MCPClient) Close() error {
	if c.stdin != nil {
		_ = c.stdin.Close()
	}

	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}

	return nil
}

// GetStderrOutput reads any stderr output from the server.
func (c *MCPClient) GetStderrOutput() string {
	var buf bytes.Buffer
	for c.stderr.Scan() {
		buf.WriteString(c.stderr.Text())
		buf.WriteString("\n")
	}

	return buf.String()
}

// Helper function to get OASF schema and validate it.
func getSchemaAndValidate(client *MCPClient, version string, requestID int) {
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "agntcy_oasf_get_schema",
			"arguments": map[string]interface{}{
				"version": version,
			},
		},
		ID: requestID,
	}

	resp, err := client.SendRequest(req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(resp.Error).To(gomega.BeNil())

	// Parse result
	var result map[string]interface{}

	err = json.Unmarshal(resp.Result, &result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	content, ok := result["content"].([]interface{})
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(content).To(gomega.HaveLen(1))

	output, ok := content[0].(map[string]interface{})
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(output["type"]).To(gomega.Equal("text"))

	textOutput, ok := output["text"].(string)
	gomega.Expect(ok).To(gomega.BeTrue())

	var toolOutput map[string]interface{}

	err = json.Unmarshal([]byte(textOutput), &toolOutput)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Expect(toolOutput["version"]).To(gomega.Equal(version))
	gomega.Expect(toolOutput["schema"]).NotTo(gomega.BeEmpty())

	// Verify it's valid JSON
	schemaStr, ok := toolOutput["schema"].(string)
	gomega.Expect(ok).To(gomega.BeTrue())

	var schema map[string]interface{}

	err = json.Unmarshal([]byte(schemaStr), &schema)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(schema).To(gomega.HaveKey("$defs"))
}

// Helper function to validate a record and parse the output.
func validateRecordAndParseOutput(client *MCPClient, recordJSON string, requestID int) map[string]interface{} {
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "agntcy_oasf_validate_record",
			"arguments": map[string]interface{}{
				"record_json": recordJSON,
			},
		},
		ID: requestID,
	}

	resp, err := client.SendRequest(req)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	gomega.Expect(resp.Error).To(gomega.BeNil())

	var result map[string]interface{}

	err = json.Unmarshal(resp.Result, &result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	content, ok := result["content"].([]interface{})
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(content).To(gomega.HaveLen(1))

	output, ok := content[0].(map[string]interface{})
	gomega.Expect(ok).To(gomega.BeTrue())
	gomega.Expect(output["type"]).To(gomega.Equal("text"))

	textOutput, ok := output["text"].(string)
	gomega.Expect(ok).To(gomega.BeTrue())

	var toolOutput map[string]interface{}

	err = json.Unmarshal([]byte(textOutput), &toolOutput)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return toolOutput
}

var _ = ginkgo.Describe("MCP Server Protocol Tests", func() {
	var client *MCPClient
	var mcpDir string

	ginkgo.BeforeEach(func() {
		// Get the MCP directory (relative to e2e/mcp)
		repoRoot := filepath.Join("..", "..")
		mcpDir = filepath.Join(repoRoot, "mcp")

		// Start MCP server using go run
		var err error
		client, err = NewMCPClient(mcpDir)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
	})

	ginkgo.AfterEach(func() {
		if client != nil {
			client.Close()
		}
	})

	ginkgo.Context("MCP Initialization", func() {
		ginkgo.It("should successfully initialize with proper capabilities", func() {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"clientInfo": map[string]string{
						"name":    "e2e-test-client",
						"version": "1.0.0",
					},
					"capabilities": map[string]interface{}{},
				},
				ID: 1,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Verify server info
			serverInfo, ok := result["serverInfo"].(map[string]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(serverInfo["name"]).To(gomega.Equal("dir-mcp-server"))
			gomega.Expect(serverInfo["version"]).To(gomega.Equal("v0.1.0"))

			// Verify capabilities
			capabilities, ok := result["capabilities"].(map[string]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(capabilities).To(gomega.HaveKey("tools"))

			ginkgo.GinkgoWriter.Printf("Server initialized successfully: %s %s\n",
				serverInfo["name"], serverInfo["version"])
		})

		ginkgo.It("should send initialized notification", func() {
			// First initialize
			initReq := MCPRequest{
				JSONRPC: "2.0",
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"clientInfo": map[string]string{
						"name":    "e2e-test-client",
						"version": "1.0.0",
					},
					"capabilities": map[string]interface{}{},
				},
				ID: 1,
			}

			resp, err := client.SendRequest(initReq)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Send initialized notification (no response expected)
			notifReq := MCPRequest{
				JSONRPC: "2.0",
				Method:  "initialized",
				Params:  map[string]interface{}{},
			}

			notifBytes, err := json.Marshal(notifReq)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			_, err = client.stdin.Write(append(notifBytes, '\n'))
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			ginkgo.GinkgoWriter.Println("Initialized notification sent successfully")
		})
	})

	ginkgo.Context("Tools Listing and Calling", func() {
		ginkgo.BeforeEach(func() {
			// Initialize session
			initReq := MCPRequest{
				JSONRPC: "2.0",
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"clientInfo": map[string]string{
						"name":    "e2e-test-client",
						"version": "1.0.0",
					},
					"capabilities": map[string]interface{}{},
				},
				ID: 1,
			}

			resp, err := client.SendRequest(initReq)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())
		})

		ginkgo.It("should list all available tools", func() {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "tools/list",
				Params:  map[string]interface{}{},
				ID:      2,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			tools, ok := result["tools"].([]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(tools).To(gomega.HaveLen(4))

			// Verify tool names
			toolNames := make(map[string]bool)
			for _, tool := range tools {
				t, ok := tool.(map[string]interface{})
				gomega.Expect(ok).To(gomega.BeTrue())

				name, ok := t["name"].(string)
				gomega.Expect(ok).To(gomega.BeTrue())

				toolNames[name] = true
				ginkgo.GinkgoWriter.Printf("  - %s: %s\n", t["name"], t["description"])
			}

			gomega.Expect(toolNames).To(gomega.HaveKey("agntcy_oasf_list_versions"))
			gomega.Expect(toolNames).To(gomega.HaveKey("agntcy_oasf_get_schema"))
			gomega.Expect(toolNames).To(gomega.HaveKey("agntcy_oasf_validate_record"))
			gomega.Expect(toolNames).To(gomega.HaveKey("agntcy_dir_push_record"))

			ginkgo.GinkgoWriter.Println("All tools listed successfully")
		})

		ginkgo.It("should validate a valid 0.7.0 record", func() {
			recordJSON := string(testdata.ExpectedRecordV070JSON)
			toolOutput := validateRecordAndParseOutput(client, recordJSON, 4)

			gomega.Expect(toolOutput["valid"]).To(gomega.BeTrue())
			gomega.Expect(toolOutput["schema_version"]).To(gomega.Equal("0.7.0"))

			ginkgo.GinkgoWriter.Println("Record validated successfully")
		})

		ginkgo.It("should validate a valid 0.3.1 record", func() {
			recordJSON := string(testdata.ExpectedRecordV031JSON)
			toolOutput := validateRecordAndParseOutput(client, recordJSON, 5)

			gomega.Expect(toolOutput["valid"]).To(gomega.BeTrue())
			gomega.Expect(toolOutput["schema_version"]).To(gomega.Equal("0.3.1"))

			ginkgo.GinkgoWriter.Println("0.3.1 record validated successfully")
		})

		ginkgo.It("should validate a valid 0.8.0 record", func() {
			recordJSON := string(testdata.ExpectedRecordV080JSON)
			toolOutput := validateRecordAndParseOutput(client, recordJSON, 6)

			gomega.Expect(toolOutput["valid"]).To(gomega.BeTrue())
			gomega.Expect(toolOutput["schema_version"]).To(gomega.Equal("0.8.0"))

			ginkgo.GinkgoWriter.Println("0.8.0 record validated successfully")
		})

		ginkgo.It("should return validation errors for invalid record", func() {
			invalidJSON := `{
			"name": "test-agent",
			"version": "1.0.0",
			"schema_version": "0.7.0",
			"description": "Test",
			"authors": ["Test"],
			"created_at": "2025-01-01T00:00:00Z"
		}`

			toolOutput := validateRecordAndParseOutput(client, invalidJSON, 7)

			gomega.Expect(toolOutput["valid"]).To(gomega.BeFalse())
			gomega.Expect(toolOutput["validation_errors"]).NotTo(gomega.BeEmpty())

			errors, ok := toolOutput["validation_errors"].([]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			ginkgo.GinkgoWriter.Printf("Validation errors returned: %v\n", errors)
		})

		ginkgo.It("should push a valid record to Directory server", func() {
			recordJSON := string(testdata.ExpectedRecordV070JSON)

			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "agntcy_dir_push_record",
					"arguments": map[string]interface{}{
						"record_json": recordJSON,
					},
				},
				ID: 8,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			var result map[string]interface{}

			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			content, ok := result["content"].([]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(content).To(gomega.HaveLen(1))

			output, ok := content[0].(map[string]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(output["type"]).To(gomega.Equal("text"))

			textOutput, ok := output["text"].(string)
			gomega.Expect(ok).To(gomega.BeTrue())

			var toolOutput map[string]interface{}

			err = json.Unmarshal([]byte(textOutput), &toolOutput)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			// Check for errors first
			if errorMsg, hasError := toolOutput["error_message"]; hasError && errorMsg != nil && errorMsg != "" {
				ginkgo.GinkgoWriter.Printf("Tool returned error: %v\n", errorMsg)
				gomega.Expect(errorMsg).To(gomega.BeEmpty(), "Push should succeed without errors")
			}

			// Verify the push response
			gomega.Expect(toolOutput["cid"]).NotTo(gomega.BeEmpty())
			gomega.Expect(toolOutput["server_address"]).NotTo(gomega.BeEmpty())

			cid, ok := toolOutput["cid"].(string)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(cid).To(gomega.HavePrefix("ba")) // CIDv1 starts with 'ba'
			gomega.Expect(len(cid)).To(gomega.BeNumerically(">", 10))

			serverAddress, ok := toolOutput["server_address"].(string)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(serverAddress).To(gomega.Equal("0.0.0.0:8888"))

			ginkgo.GinkgoWriter.Printf("Record pushed successfully with CID: %s to server: %s\n", cid, serverAddress)
		})
	})

	ginkgo.Context("Schema Tools", func() {
		ginkgo.BeforeEach(func() {
			// Initialize session
			initReq := MCPRequest{
				JSONRPC: "2.0",
				Method:  "initialize",
				Params: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"clientInfo": map[string]string{
						"name":    "e2e-test-client",
						"version": "1.0.0",
					},
					"capabilities": map[string]interface{}{},
				},
				ID: 1,
			}

			resp, err := client.SendRequest(initReq)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())
		})

		ginkgo.It("should list available schema versions", func() {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name":      "agntcy_oasf_list_versions",
					"arguments": map[string]interface{}{},
				},
				ID: 2,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			content, ok := result["content"].([]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(content).To(gomega.HaveLen(1))

			output, ok := content[0].(map[string]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(output["type"]).To(gomega.Equal("text"))

			textOutput, ok := output["text"].(string)
			gomega.Expect(ok).To(gomega.BeTrue())

			var toolOutput map[string]interface{}
			err = json.Unmarshal([]byte(textOutput), &toolOutput)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			availableVersions, ok := toolOutput["available_versions"].([]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(availableVersions).To(gomega.ContainElement("0.3.1"))
			gomega.Expect(availableVersions).To(gomega.ContainElement("0.7.0"))

			count, ok := toolOutput["count"].(float64)
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(count).To(gomega.BeNumerically(">=", 2))

			ginkgo.GinkgoWriter.Printf("Available versions: %v (count: %v)\n", availableVersions, count)
		})

		ginkgo.It("should get OASF 0.7.0 schema", func() {
			getSchemaAndValidate(client, "0.7.0", 3)
			ginkgo.GinkgoWriter.Println("OASF 0.7.0 schema retrieved successfully")
		})

		ginkgo.It("should get OASF 0.3.1 schema", func() {
			getSchemaAndValidate(client, "0.3.1", 4)
			ginkgo.GinkgoWriter.Println("OASF 0.3.1 schema retrieved successfully")
		})

		ginkgo.It("should return error for invalid schema version", func() {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "agntcy_oasf_get_schema",
					"arguments": map[string]interface{}{
						"version": "999.999.999",
					},
				},
				ID: 5,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			content, ok := result["content"].([]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(content).To(gomega.HaveLen(1))

			output, ok := content[0].(map[string]interface{})
			gomega.Expect(ok).To(gomega.BeTrue())

			textOutput, ok := output["text"].(string)
			gomega.Expect(ok).To(gomega.BeTrue())

			var toolOutput map[string]interface{}
			err = json.Unmarshal([]byte(textOutput), &toolOutput)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(toolOutput["error_message"]).NotTo(gomega.BeEmpty())
			gomega.Expect(toolOutput["available_versions"]).NotTo(gomega.BeEmpty())

			ginkgo.GinkgoWriter.Printf("Error message: %v\n", toolOutput["error_message"])
		})
	})
})
