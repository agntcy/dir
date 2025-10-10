// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/agntcy/dir/e2e/shared/testdata"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

// MCPRequest represents a JSON-RPC 2.0 request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// MCPResponse represents a JSON-RPC 2.0 response
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents a JSON-RPC 2.0 error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPClient manages the MCP server process and communication
type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr *bufio.Scanner
}

// NewMCPClient starts an MCP server and returns a client to communicate with it
func NewMCPClient(binaryPath string) (*MCPClient, error) {
	cmd := exec.Command(binaryPath)

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

// SendRequest sends a JSON-RPC request and returns the response
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
		return nil, fmt.Errorf("no response received")
	}

	// Parse response
	var resp MCPResponse
	if err := json.Unmarshal(c.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// Close stops the MCP server and cleans up
func (c *MCPClient) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
		c.cmd.Wait()
	}
	return nil
}

// GetStderrOutput reads any stderr output from the server
func (c *MCPClient) GetStderrOutput() string {
	var buf bytes.Buffer
	for c.stderr.Scan() {
		buf.WriteString(c.stderr.Text())
		buf.WriteString("\n")
	}
	return buf.String()
}

var _ = ginkgo.Describe("MCP Server Protocol Tests", func() {
	var client *MCPClient
	var mcpBinaryPath string

	ginkgo.BeforeEach(func() {
		// Get the MCP binary path (relative to e2e/mcp)
		repoRoot := filepath.Join("..", "..")
		mcpBinaryPath = filepath.Join(repoRoot, "mcp", "mcp")

		// Check if binary exists, if not try to build it
		if _, err := os.Stat(mcpBinaryPath); os.IsNotExist(err) {
			ginkgo.GinkgoWriter.Printf("MCP binary not found at %s, building it...\n", mcpBinaryPath)
			buildCmd := exec.Command("go", "build", "-o", "mcp")
			buildCmd.Dir = filepath.Join(repoRoot, "mcp")
			if output, err := buildCmd.CombinedOutput(); err != nil {
				ginkgo.Fail(fmt.Sprintf("Failed to build MCP binary: %v\n%s", err, output))
			}
		}

		// Start MCP server
		var err error
		client, err = NewMCPClient(mcpBinaryPath)
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
			serverInfo := result["serverInfo"].(map[string]interface{})
			gomega.Expect(serverInfo["name"]).To(gomega.Equal("dir-mcp-server"))
			gomega.Expect(serverInfo["version"]).To(gomega.Equal("v0.1.0"))

			// Verify capabilities
			capabilities := result["capabilities"].(map[string]interface{})
			gomega.Expect(capabilities).To(gomega.HaveKey("tools"))
			gomega.Expect(capabilities).To(gomega.HaveKey("resources"))

			// Verify resource capabilities
			resources := capabilities["resources"].(map[string]interface{})
			gomega.Expect(resources["listChanged"]).To(gomega.BeTrue())

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

			tools := result["tools"].([]interface{})
			gomega.Expect(tools).To(gomega.HaveLen(1))

			// Verify tool names
			toolNames := make(map[string]bool)
			for _, tool := range tools {
				t := tool.(map[string]interface{})
				toolNames[t["name"].(string)] = true
				ginkgo.GinkgoWriter.Printf("  - %s: %s\n", t["name"], t["description"])
			}

			gomega.Expect(toolNames).To(gomega.HaveKey("agntcy_oasf_validate_record"))

			ginkgo.GinkgoWriter.Println("All tools listed successfully")
		})

		ginkgo.It("should validate a valid 0.7.0 record", func() {
			recordJSON := string(testdata.ExpectedRecordV070JSON)

			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "agntcy_oasf_validate_record",
					"arguments": map[string]interface{}{
						"record_json": recordJSON,
					},
				},
				ID: 4,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			content := result["content"].([]interface{})
			gomega.Expect(content).To(gomega.HaveLen(1))

			output := content[0].(map[string]interface{})
			gomega.Expect(output["type"]).To(gomega.Equal("text"))

			// Parse the text output
			var toolOutput map[string]interface{}
			err = json.Unmarshal([]byte(output["text"].(string)), &toolOutput)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(toolOutput["valid"]).To(gomega.BeTrue())
			gomega.Expect(toolOutput["schema_version"]).To(gomega.Equal("0.7.0"))

			ginkgo.GinkgoWriter.Println("Record validated successfully")
		})

		ginkgo.It("should validate a valid 0.3.1 record", func() {
			recordJSON := string(testdata.ExpectedRecordV031JSON)

			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "agntcy_oasf_validate_record",
					"arguments": map[string]interface{}{
						"record_json": recordJSON,
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

			content := result["content"].([]interface{})
			gomega.Expect(content).To(gomega.HaveLen(1))

			output := content[0].(map[string]interface{})
			gomega.Expect(output["type"]).To(gomega.Equal("text"))

			// Parse the text output
			var toolOutput map[string]interface{}
			err = json.Unmarshal([]byte(output["text"].(string)), &toolOutput)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(toolOutput["valid"]).To(gomega.BeTrue())
			gomega.Expect(toolOutput["schema_version"]).To(gomega.Equal("0.3.1"))

			ginkgo.GinkgoWriter.Println("0.3.1 record validated successfully")
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

			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "agntcy_oasf_validate_record",
					"arguments": map[string]interface{}{
						"record_json": invalidJSON,
					},
				},
				ID: 6,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			content := result["content"].([]interface{})
			output := content[0].(map[string]interface{})

			var toolOutput map[string]interface{}
			err = json.Unmarshal([]byte(output["text"].(string)), &toolOutput)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			gomega.Expect(toolOutput["valid"]).To(gomega.BeFalse())
			gomega.Expect(toolOutput["validation_errors"]).NotTo(gomega.BeEmpty())

			errors := toolOutput["validation_errors"].([]interface{})
			ginkgo.GinkgoWriter.Printf("Validation errors returned: %v\n", errors)
		})
	})

	ginkgo.Context("Resources Listing and Reading", func() {
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

		ginkgo.It("should list all available resources", func() {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "resources/list",
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

			resources := result["resources"].([]interface{})
			gomega.Expect(resources).To(gomega.HaveLen(2))

			// Verify resource URIs
			resourceURIs := make(map[string]bool)
			for _, resource := range resources {
				r := resource.(map[string]interface{})
				resourceURIs[r["uri"].(string)] = true
				ginkgo.GinkgoWriter.Printf("  - %s: %s\n", r["name"], r["description"])
			}

			gomega.Expect(resourceURIs).To(gomega.HaveKey("agntcy://oasf/schema/0.3.1"))
			gomega.Expect(resourceURIs).To(gomega.HaveKey("agntcy://oasf/schema/0.7.0"))

			ginkgo.GinkgoWriter.Println("All resources listed successfully")
		})

		ginkgo.It("should read OASF 0.7.0 schema resource", func() {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "resources/read",
				Params: map[string]interface{}{
					"uri": "agntcy://oasf/schema/0.7.0",
				},
				ID: 3,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			contents := result["contents"].([]interface{})
			gomega.Expect(contents).To(gomega.HaveLen(1))

			content := contents[0].(map[string]interface{})
			gomega.Expect(content["uri"]).To(gomega.Equal("agntcy://oasf/schema/0.7.0"))
			gomega.Expect(content["mimeType"]).To(gomega.Equal("application/json"))
			gomega.Expect(content["text"]).NotTo(gomega.BeEmpty())

			// Verify it's valid JSON
			var schema map[string]interface{}
			err = json.Unmarshal([]byte(content["text"].(string)), &schema)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(schema).To(gomega.HaveKey("$defs"))

			ginkgo.GinkgoWriter.Println("OASF 0.7.0 schema resource read successfully")
		})

		ginkgo.It("should read OASF 0.3.1 schema resource", func() {
			req := MCPRequest{
				JSONRPC: "2.0",
				Method:  "resources/read",
				Params: map[string]interface{}{
					"uri": "agntcy://oasf/schema/0.3.1",
				},
				ID: 4,
			}

			resp, err := client.SendRequest(req)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(resp.Error).To(gomega.BeNil())

			// Parse result
			var result map[string]interface{}
			err = json.Unmarshal(resp.Result, &result)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())

			contents := result["contents"].([]interface{})
			gomega.Expect(contents).To(gomega.HaveLen(1))

			content := contents[0].(map[string]interface{})
			gomega.Expect(content["uri"]).To(gomega.Equal("agntcy://oasf/schema/0.3.1"))
			gomega.Expect(content["mimeType"]).To(gomega.Equal("application/json"))
			gomega.Expect(content["text"]).NotTo(gomega.BeEmpty())

			// Verify it's valid JSON
			var schema map[string]interface{}
			err = json.Unmarshal([]byte(content["text"].(string)), &schema)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			gomega.Expect(schema).To(gomega.HaveKey("$defs"))

			ginkgo.GinkgoWriter.Println("OASF 0.3.1 schema resource read successfully")
		})
	})
})
