// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package local

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	gnet "github.com/shirou/gopsutil/v4/net"
	gprocess "github.com/shirou/gopsutil/v4/process"
)

// Process represents a local process with its details and associated resources.
type Process struct {
	Pid       int32
	ParentPid int32
	Name      string
	CreatedAt string
	Username  string
	Exe       string
	Cmdline   string
	Env       map[string]string
	Ports     []uint32
	Addresses []string
	Pipes     []string
	Sockets   []string
}

// GetProcess retrieves detailed information about a process given its PID, including metadata, environment variables
// and associated network and file descriptor details. It uses gopsutil to fetch process information.
//
// Process data discovery is done in best-effort manner and returned details may be incomplete/unreliable.
//
// TODO: network fetching is disabled due to package reliability.
// TODO: investigate how to fetch networking data for a process better.
func GetProcess(ctx context.Context, pid int32) (*Process, error) {
	// Fetch process details
	proc, err := gprocess.NewProcessWithContext(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to create process for pid %d: %w", pid, err)
	}

	// Extract common meta
	exe, _ := proc.ExeWithContext(ctx)
	cmdline, _ := proc.CmdlineWithContext(ctx)
	username, _ := proc.UsernameWithContext(ctx)
	ppid, _ := proc.PpidWithContext(ctx)
	createdAt, _ := proc.CreateTimeWithContext(ctx)

	// Extract name
	name, _ := proc.NameWithContext(ctx)
	if name == "" && exe != "" {
		name = filepath.Base(exe)
	}

	if name == "" {
		name = strconv.Itoa(int(pid))
	}

	// Extract process environment variables
	procEnv := make(map[string]string)

	_environ, _ := proc.EnvironWithContext(ctx)
	for _, entry := range _environ {
		key, value, found := strings.Cut(entry, "=")
		if !found || key == "" {
			continue
		}

		procEnv[key] = value
	}

	// TODO: reenable network and file descriptor details once fetching is more reliable
	// // Extract network details
	// ports, addresses := getNetworkDetails(ctx, pid)

	// // Extract file descriptor details
	// pipes, sockets := getPipeSocketDetails(ctx, proc)

	return &Process{
		Pid:       pid,
		ParentPid: ppid,
		Name:      name,
		CreatedAt: time.UnixMilli(createdAt).UTC().Format(time.RFC3339),
		Username:  username,
		Exe:       exe,
		Cmdline:   cmdline,
		Env:       procEnv,
	}, nil
}

// getNetworkDetails retrieves network connections for the given PID and categorizes them into ports and addresses.
//
//nolint:unused
func getNetworkDetails(ctx context.Context, pid int32) (map[uint32]struct{}, map[string]struct{}) {
	// Get all network connections for the process
	conns, err := gnet.ConnectionsPidWithContext(ctx, "all", pid)
	if err != nil {
		return nil, nil
	}

	portSet := make(map[uint32]struct{})
	addressSet := make(map[string]struct{})

	// Process connections to extract listening ports and associated addresses
	for _, conn := range conns {
		// Get connection status
		status := strings.ToUpper(conn.Status)
		isListening := status == "LISTEN" || status == "NONE"

		// Only consider connections that are in a listening state
		if !isListening {
			continue
		}

		// Extract port and address information
		if port := conn.Laddr.Port; port > 0 {
			portSet[port] = struct{}{}
		}

		if ip := strings.TrimSpace(conn.Laddr.IP); ip != "" {
			addressSet[ip] = struct{}{}
		}
	}

	// If no specific listening addresses were found but there are listening ports, assume localhost.
	if len(addressSet) == 0 && len(portSet) > 0 {
		addressSet["127.0.0.1"] = struct{}{}
	}

	return portSet, addressSet
}

// getPipeSocketDetails retrieves open file descriptors for the given process and
// categorizes them into pipes and sockets based on their paths.
//
//nolint:unused
func getPipeSocketDetails(ctx context.Context, proc *gprocess.Process) (map[string]struct{}, map[string]struct{}) {
	// Get all open files for the process
	files, err := proc.OpenFilesWithContext(ctx)
	if err != nil {
		return nil, nil
	}

	pipeSet := make(map[string]struct{})
	socketSet := make(map[string]struct{})

	// Analyze file paths to categorize them as pipes or sockets
	for _, file := range files {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}

		lowerPath := strings.ToLower(path)
		switch {
		case strings.Contains(lowerPath, "pipe") || strings.Contains(lowerPath, "fifo"):
			pipeSet[path] = struct{}{}
		case strings.Contains(lowerPath, "socket"):
			socketSet[path] = struct{}{}
		}
	}

	return pipeSet, socketSet
}
