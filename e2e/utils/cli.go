// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	clicmd "github.com/agntcy/dir/cli/cmd"
	"github.com/onsi/gomega"
)

const (
	// DefaultCommandTimeout is the default timeout for CLI command execution.
	DefaultCommandTimeout = 30 * time.Second
	// PollingInterval is the interval for Eventually polling operations.
	PollingInterval = 5 * time.Second
)

// CLI provides a fluent interface for executing CLI commands in tests.
type CLI struct{}

// NewCLI creates a new CLI test helper.
func NewCLI() *CLI {
	return &CLI{}
}

// Command creates a new command builder.
func (c *CLI) Command(name string) *CommandBuilder {
	return &CommandBuilder{
		command: name,
		timeout: DefaultCommandTimeout,
	}
}

// Convenience methods for common commands.
func (c *CLI) Push(path string) *CommandBuilder {
	return c.Command("push").WithArgs(path)
}

func (c *CLI) Pull(cid string) *CommandBuilder {
	return c.Command("pull").WithArgs(cid)
}

func (c *CLI) Delete(cid string) *CommandBuilder {
	return c.Command("delete").WithArgs(cid)
}

func (c *CLI) Search() *SearchBuilder {
	return &SearchBuilder{
		CommandBuilder: c.Command("search"),
		queries:        make(map[string]string),
	}
}

func (c *CLI) Sign(recordPath, keyPath string) *CommandBuilder {
	return c.Command("sign").WithArgs(recordPath, "--key", keyPath)
}

func (c *CLI) Publish(cid string) *CommandBuilder {
	return c.Command("publish").WithArgs(cid)
}

func (c *CLI) List() *ListBuilder {
	return &ListBuilder{
		CommandBuilder: c.Command("list"),
	}
}

func (c *CLI) Verify(recordPath, signaturePath, keyPath string) *CommandBuilder {
	return c.Command("verify").WithArgs(recordPath, signaturePath, "--key", keyPath)
}

// Network commands.
func (c *CLI) Network() *NetworkCommands {
	return &NetworkCommands{cli: c}
}

type NetworkCommands struct {
	cli *CLI
}

func (n *NetworkCommands) Info(keyPath string) *CommandBuilder {
	return n.cli.Command("network").WithArgs("info", keyPath)
}

func (n *NetworkCommands) Init() *CommandBuilder {
	return n.cli.Command("network").WithArgs("init")
}

// Sync commands.
func (c *CLI) Sync() *SyncCommands {
	return &SyncCommands{cli: c}
}

type SyncCommands struct {
	cli *CLI
}

func (s *SyncCommands) Create(url string) *CommandBuilder {
	return s.cli.Command("sync").WithArgs("create", url)
}

func (s *SyncCommands) List() *CommandBuilder {
	return s.cli.Command("sync").WithArgs("list")
}

func (s *SyncCommands) Status(syncID string) *CommandBuilder {
	return s.cli.Command("sync").WithArgs("status", syncID)
}

func (s *SyncCommands) Delete(syncID string) *CommandBuilder {
	return s.cli.Command("sync").WithArgs("delete", syncID)
}

// CommandBuilder provides a fluent interface for building and executing commands.
type CommandBuilder struct {
	command    string
	args       []string
	serverAddr string
	expectErr  bool
	timeout    time.Duration
	outputFile string
}

func (c *CommandBuilder) WithArgs(args ...string) *CommandBuilder {
	c.args = append(c.args, args...)

	return c
}

func (c *CommandBuilder) OnServer(addr string) *CommandBuilder {
	c.serverAddr = addr

	return c
}

func (c *CommandBuilder) WithTimeout(timeout time.Duration) *CommandBuilder {
	c.timeout = timeout

	return c
}

func (c *CommandBuilder) WithOutput(path string) *CommandBuilder {
	c.outputFile = path

	return c
}

func (c *CommandBuilder) ExpectError() *CommandBuilder {
	c.expectErr = true

	return c
}

// Execute runs the command and returns output and error.
func (c *CommandBuilder) Execute() (string, error) {
	args := append([]string{c.command}, c.args...)

	if c.serverAddr != "" {
		args = append(args, "--server-addr", c.serverAddr)
	}

	if c.outputFile != "" {
		args = append(args, "--output", c.outputFile)
	}

	var outputBuffer bytes.Buffer

	cmd := clicmd.RootCmd
	cmd.SetOut(&outputBuffer)
	cmd.SetArgs(args)

	err := cmd.Execute()
	output := strings.TrimSpace(outputBuffer.String())

	if err != nil {
		return output, fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}

// ShouldSucceed executes the command and expects success.
func (c *CommandBuilder) ShouldSucceed() string {
	output, err := c.Execute()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(),
		fmt.Sprintf("Command '%s %s' should succeed", c.command, strings.Join(c.args, " ")))

	return output
}

// ShouldFail executes the command and expects failure.
func (c *CommandBuilder) ShouldFail() error {
	_, err := c.Execute()
	gomega.Expect(err).To(gomega.HaveOccurred(),
		fmt.Sprintf("Command '%s %s' should fail", c.command, strings.Join(c.args, " ")))

	return err
}

// ShouldReturn executes the command and expects specific output.
func (c *CommandBuilder) ShouldReturn(expected string) {
	output := c.ShouldSucceed()
	gomega.Expect(output).To(gomega.Equal(expected))
}

// ShouldContain executes the command and expects output to contain substring.
func (c *CommandBuilder) ShouldContain(substring string) string {
	output := c.ShouldSucceed()
	gomega.Expect(output).To(gomega.ContainSubstring(substring))

	return output
}

// ShouldEventuallyContain polls the command until output contains substring.
func (c *CommandBuilder) ShouldEventuallyContain(substring string, timeout time.Duration) string {
	var finalOutput string

	gomega.Eventually(func() string {
		output, err := c.Execute()
		if err != nil {
			return ""
		}
		finalOutput = output

		return output
	}, timeout, PollingInterval).Should(gomega.ContainSubstring(substring))

	return finalOutput
}

// ShouldEventuallySucceed polls the command until it succeeds.
func (c *CommandBuilder) ShouldEventuallySucceed(timeout time.Duration) string {
	var finalOutput string

	gomega.Eventually(func() error {
		output, err := c.Execute()
		finalOutput = output

		return err
	}, timeout, PollingInterval).Should(gomega.Succeed())

	return finalOutput
}

// SearchBuilder extends CommandBuilder with search-specific methods.
type SearchBuilder struct {
	*CommandBuilder
	queries map[string]string
	limit   int
	offset  int
}

func (s *SearchBuilder) WithQuery(key, value string) *SearchBuilder {
	s.queries[key] = value

	return s
}

func (s *SearchBuilder) WithLimit(limit int) *SearchBuilder {
	s.limit = limit

	return s
}

func (s *SearchBuilder) WithOffset(offset int) *SearchBuilder {
	s.offset = offset

	return s
}

func (s *SearchBuilder) Execute() (string, error) {
	// Build search arguments
	for key, value := range s.queries {
		s.args = append(s.args, "--query", fmt.Sprintf(`%s=%s`, key, value))
	}

	if s.limit > 0 {
		s.args = append(s.args, "--limit", strconv.Itoa(s.limit))
	}

	if s.offset > 0 {
		s.args = append(s.args, "--offset", strconv.Itoa(s.offset))
	}

	return s.CommandBuilder.Execute()
}

func (s *SearchBuilder) ShouldSucceed() string {
	output, err := s.Execute()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return output
}

func (s *SearchBuilder) ShouldReturn(expectedCID string) {
	output := s.ShouldSucceed()
	gomega.Expect(output).To(gomega.Equal(expectedCID))
}

// ListBuilder extends CommandBuilder with list-specific methods.
type ListBuilder struct {
	*CommandBuilder
}

func (l *ListBuilder) WithDigest(digest string) *ListBuilder {
	l.args = append(l.args, "--digest", digest)

	return l
}

func (l *ListBuilder) WithSkill(skill string) *ListBuilder {
	l.args = append(l.args, fmt.Sprintf(`"%s"`, skill))

	return l
}
