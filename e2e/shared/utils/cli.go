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
	// PublishProcessingDelay is the delay to allow asynchronous publish operations to complete.
	PublishProcessingDelay = 15 * time.Second
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
		CommandBuilder:   c.Command("search"),
		names:            []string{},
		versions:         []string{},
		skillIDs:         []string{},
		skillNames:       []string{},
		locators:         []string{},
		modules:          []string{},
		domainIDs:        []string{},
		domains:          []string{},
		outputFormatArgs: []string{},
		limit:            0,
		offset:           0,
	}
}

func (c *CLI) Sign(recordCID, keyPath string) *CommandBuilder {
	return c.Command("sign").WithArgs(recordCID, "--key", keyPath)
}

// Routing commands - all routing operations are now under the routing subcommand.
func (c *CLI) Routing() *RoutingCommands {
	return &RoutingCommands{cli: c}
}

type RoutingCommands struct {
	cli *CLI
}

func (r *RoutingCommands) Publish(cid string) *CommandBuilder {
	return r.cli.Command("routing").WithArgs("publish", cid)
}

func (r *RoutingCommands) Unpublish(cid string) *CommandBuilder {
	return r.cli.Command("routing").WithArgs("unpublish", cid)
}

func (r *RoutingCommands) List() *RoutingListBuilder {
	return &RoutingListBuilder{
		CommandBuilder: r.cli.Command("routing").WithArgs("list"),
	}
}

func (r *RoutingCommands) Search() *RoutingSearchBuilder {
	return &RoutingSearchBuilder{
		CommandBuilder: r.cli.Command("routing").WithArgs("search"),
	}
}

func (r *RoutingCommands) Info() *CommandBuilder {
	return r.cli.Command("routing").WithArgs("info")
}

func (r *RoutingCommands) WithArgs(args ...string) *CommandBuilder {
	return r.cli.Command("routing").WithArgs(args...)
}

func (c *CLI) Verify(recordCID string) *CommandBuilder {
	return c.Command("verify").WithArgs(recordCID)
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

func (s *SyncCommands) CreateFromStdin(input string) *StdinCommandBuilder {
	return &StdinCommandBuilder{
		CommandBuilder: s.cli.Command("sync").WithArgs("create", "--stdin"),
		stdinInput:     input,
	}
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
	command     string
	args        []string
	serverAddr  string
	expectErr   bool
	timeout     time.Duration
	outputFile  string
	suppressErr bool
}

// StdinCommandBuilder extends CommandBuilder to handle stdin input.
type StdinCommandBuilder struct {
	*CommandBuilder
	stdinInput string
}

// OnServer sets the server address for StdinCommandBuilder.
func (s *StdinCommandBuilder) OnServer(addr string) *StdinCommandBuilder {
	s.CommandBuilder.OnServer(addr)

	return s
}

// WithTimeout sets the timeout for StdinCommandBuilder.
func (s *StdinCommandBuilder) WithTimeout(timeout time.Duration) *StdinCommandBuilder {
	s.CommandBuilder.WithTimeout(timeout)

	return s
}

// Execute runs the command with stdin input and returns output and error.
func (s *StdinCommandBuilder) Execute() (string, error) {
	args := append([]string{s.command}, s.args...)

	if s.serverAddr != "" {
		args = append(args, "--server-addr", s.serverAddr)
	}

	if s.outputFile != "" {
		args = append(args, "--output", s.outputFile)
	}

	var outputBuffer bytes.Buffer

	var errorBuffer bytes.Buffer

	cmd := clicmd.RootCmd

	// Store original stdin to restore later
	originalIn := cmd.InOrStdin()

	cmd.SetOut(&outputBuffer)

	if s.suppressErr {
		cmd.SetErr(&errorBuffer) // Capture stderr to suppress it
	}

	// Set stdin input
	cmd.SetIn(strings.NewReader(s.stdinInput))
	cmd.SetArgs(args)

	err := cmd.Execute()
	output := strings.TrimSpace(outputBuffer.String())

	// Restore original stdin
	cmd.SetIn(originalIn)

	if err != nil {
		return output, fmt.Errorf("command execution failed: %w", err)
	}

	return output, nil
}

// ShouldSucceed executes the command with stdin and expects success.
func (s *StdinCommandBuilder) ShouldSucceed() string {
	output, err := s.Execute()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(),
		fmt.Sprintf("Command '%s %s' should succeed", s.command, strings.Join(s.args, " ")))

	return output
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

func (c *CommandBuilder) SuppressStderr() *CommandBuilder {
	c.suppressErr = true

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

	var errorBuffer bytes.Buffer

	cmd := clicmd.RootCmd
	cmd.SetOut(&outputBuffer)

	if c.suppressErr {
		cmd.SetErr(&errorBuffer) // Capture stderr to suppress it
	}

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
	// Automatically suppress stderr for expected failures to reduce noise
	c.suppressErr = true
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
	names            []string
	versions         []string
	skillIDs         []string
	skillNames       []string
	locators         []string
	modules          []string
	domainIDs        []string
	domains          []string
	outputFormatArgs []string
	limit            int
	offset           int
}

func (s *SearchBuilder) WithName(name string) *SearchBuilder {
	s.names = append(s.names, name)

	return s
}

func (s *SearchBuilder) WithVersion(version string) *SearchBuilder {
	s.versions = append(s.versions, version)

	return s
}

func (s *SearchBuilder) WithSkillID(skillID string) *SearchBuilder {
	s.skillIDs = append(s.skillIDs, skillID)

	return s
}

func (s *SearchBuilder) WithSkillName(skillName string) *SearchBuilder {
	s.skillNames = append(s.skillNames, skillName)

	return s
}

func (s *SearchBuilder) WithLocator(locator string) *SearchBuilder {
	s.locators = append(s.locators, locator)

	return s
}

func (s *SearchBuilder) WithModule(module string) *SearchBuilder {
	s.modules = append(s.modules, module)

	return s
}

func (s *SearchBuilder) WithDomainID(domainID string) *SearchBuilder {
	s.domainIDs = append(s.domainIDs, domainID)

	return s
}

func (s *SearchBuilder) WithDomain(domain string) *SearchBuilder {
	s.domains = append(s.domains, domain)

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

func (s *SearchBuilder) WithArgs(args ...string) *SearchBuilder {
	s.outputFormatArgs = append(s.outputFormatArgs, args...)

	return s
}

func (s *SearchBuilder) Execute() (string, error) {
	// Clear existing arguments to prevent accumulation between test cases
	s.args = nil

	// Build search arguments using new direct field flags
	for _, name := range s.names {
		s.args = append(s.args, "--name", name)
	}

	for _, version := range s.versions {
		s.args = append(s.args, "--version", version)
	}

	for _, skillID := range s.skillIDs {
		s.args = append(s.args, "--skill-id", skillID)
	}

	for _, skillName := range s.skillNames {
		s.args = append(s.args, "--skill", skillName)
	}

	for _, locator := range s.locators {
		s.args = append(s.args, "--locator", locator)
	}

	for _, module := range s.modules {
		s.args = append(s.args, "--module", module)
	}

	for _, domainID := range s.domainIDs {
		s.args = append(s.args, "--domain-id", domainID)
	}

	for _, domain := range s.domains {
		s.args = append(s.args, "--domain", domain)
	}

	if s.limit > 0 {
		s.args = append(s.args, "--limit", strconv.Itoa(s.limit))
	}

	if s.offset > 0 {
		s.args = append(s.args, "--offset", strconv.Itoa(s.offset))
	}

	// Append any additional args (like --output) at the end
	s.args = append(s.args, s.outputFormatArgs...)

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

// RoutingListBuilder extends CommandBuilder with routing list-specific methods.
type RoutingListBuilder struct {
	*CommandBuilder
}

func (l *RoutingListBuilder) WithCid(cid string) *RoutingListBuilder {
	l.args = append(l.args, "--cid", cid)

	return l
}

func (l *RoutingListBuilder) WithSkill(skill string) *RoutingListBuilder {
	l.args = append(l.args, "--skill", skill)

	return l
}

func (l *RoutingListBuilder) WithLocator(locator string) *RoutingListBuilder {
	l.args = append(l.args, "--locator", locator)

	return l
}

func (l *RoutingListBuilder) WithDomain(domain string) *RoutingListBuilder {
	l.args = append(l.args, "--domain", domain)

	return l
}

func (l *RoutingListBuilder) WithModule(module string) *RoutingListBuilder {
	l.args = append(l.args, "--module", module)

	return l
}

func (l *RoutingListBuilder) WithLimit(limit int) *RoutingListBuilder {
	l.args = append(l.args, "--limit", strconv.Itoa(limit))

	return l
}

// RoutingSearchBuilder extends CommandBuilder with routing search-specific methods.
type RoutingSearchBuilder struct {
	*CommandBuilder
}

func (s *RoutingSearchBuilder) WithSkill(skill string) *RoutingSearchBuilder {
	s.args = append(s.args, "--skill", skill)

	return s
}

func (s *RoutingSearchBuilder) WithLocator(locator string) *RoutingSearchBuilder {
	s.args = append(s.args, "--locator", locator)

	return s
}

func (s *RoutingSearchBuilder) WithDomain(domain string) *RoutingSearchBuilder {
	s.args = append(s.args, "--domain", domain)

	return s
}

func (s *RoutingSearchBuilder) WithModule(module string) *RoutingSearchBuilder {
	s.args = append(s.args, "--module", module)

	return s
}

func (s *RoutingSearchBuilder) WithLimit(limit int) *RoutingSearchBuilder {
	s.args = append(s.args, "--limit", strconv.Itoa(limit))

	return s
}

func (s *RoutingSearchBuilder) WithMinScore(minScore int) *RoutingSearchBuilder {
	s.args = append(s.args, "--min-score", strconv.Itoa(minScore))

	return s
}
