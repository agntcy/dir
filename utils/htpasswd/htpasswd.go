// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package htpasswd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/agntcy/dir/utils/logging"
	"golang.org/x/crypto/bcrypt"
)

var logger = logging.Logger("utils/htpasswd")

// Manager handles htpasswd file operations with thread safety.
type Manager struct {
	htpasswdPath string
	mutex        sync.RWMutex
}

// Credentials represents a username/password pair.
type Credentials struct {
	Username string
	Password string
}

// NewManager creates a new htpasswd manager with the specified file path.
// If the path doesn't exist, it will be created when credentials are added.
func NewManager(htpasswdPath string) *Manager {
	return &Manager{
		htpasswdPath: htpasswdPath,
	}
}

// GenerateCredentials creates a new username/password pair for the given node ID.
// The username is formatted as "sync-<nodeID>" and a secure random password is generated.
//
//nolint:mnd
func (m *Manager) GenerateCredentials(nodeID string) (*Credentials, error) {
	// Generate a unique username based on the requesting node ID
	username := "sync-" + nodeID

	// Generate a secure random password (128-bit)
	passwordBytes := make([]byte, 16)
	if _, err := rand.Read(passwordBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random password: %w", err)
	}

	password := hex.EncodeToString(passwordBytes)

	// Generate bcrypt hash for htpasswd
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Add user to htpasswd file
	if err := m.addUser(username, string(hashedPassword)); err != nil {
		return nil, fmt.Errorf("failed to add user to htpasswd: %w", err)
	}

	logger.Debug("Generated htpasswd credentials",
		"username", username,
		"node_id", nodeID)

	return &Credentials{
		Username: username,
		Password: password,
	}, nil
}

// AddUser adds or updates a user with a plaintext password in the htpasswd file.
func (m *Manager) AddUser(username, password string) error {
	// Generate bcrypt hash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return m.addUser(username, string(hashedPassword))
}

// RemoveUser removes a user from the htpasswd file.
func (m *Manager) RemoveUser(username string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Read existing users
	existingUsers, err := m.readUsers()
	if err != nil {
		return err
	}

	// Remove the user
	delete(existingUsers, username)

	// Write updated file
	return m.writeUsers(existingUsers)
}

// ListUsers returns a list of all usernames in the htpasswd file.
func (m *Manager) ListUsers() ([]string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	existingUsers, err := m.readUsers()
	if err != nil {
		return nil, err
	}

	users := make([]string, 0, len(existingUsers))
	for username := range existingUsers {
		users = append(users, username)
	}

	return users, nil
}

// GetPath returns the htpasswd file path.
func (m *Manager) GetPath() string {
	return m.htpasswdPath
}

// addUser adds or updates a user with a hashed password in the htpasswd file.
func (m *Manager) addUser(username, hashedPassword string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Read existing users
	existingUsers, err := m.readUsers()
	if err != nil {
		return err
	}

	// Add or update the user
	existingUsers[username] = hashedPassword

	// Write updated file
	if err := m.writeUsers(existingUsers); err != nil {
		return err
	}

	logger.Debug("Updated htpasswd file",
		"path", m.htpasswdPath,
		"username", username,
		"total_users", len(existingUsers))

	return nil
}

// readUsers reads the htpasswd file and returns a map of username to hashed password.
//
//nolint:mnd
func (m *Manager) readUsers() (map[string]string, error) {
	existingUsers := make(map[string]string)

	content, err := os.ReadFile(m.htpasswdPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return empty map
			return existingUsers, nil
		}

		return nil, fmt.Errorf("failed to read htpasswd file: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				existingUsers[parts[0]] = parts[1]
			}
		}
	}

	return existingUsers, nil
}

// writeUsers writes the users map to the htpasswd file.
//
//nolint:mnd
func (m *Manager) writeUsers(users map[string]string) error {
	// Ensure directory exists
	dir := filepath.Dir(m.htpasswdPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		// If we can't create the directory, provide a helpful error message
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied creating htpasswd directory %s: %w. "+
				"Ensure the directory exists and is writable by user 65532, or use an init container to fix permissions", dir, err)
		}

		return fmt.Errorf("failed to create htpasswd directory %s: %w", dir, err)
	}

	// Build htpasswd content
	var content strings.Builder
	for user, hash := range users {
		content.WriteString(fmt.Sprintf("%s:%s\n", user, hash))
	}

	// Write the htpasswd file
	if err := os.WriteFile(m.htpasswdPath, []byte(content.String()), 0o644); err != nil { //nolint:gosec
		// Provide helpful error message for permission issues
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied writing htpasswd file %s: %w. "+
				"Ensure the file and directory are writable by user 65532, or use an init container to fix permissions", m.htpasswdPath, err)
		}

		return fmt.Errorf("failed to write htpasswd file %s: %w", m.htpasswdPath, err)
	}

	return nil
}
