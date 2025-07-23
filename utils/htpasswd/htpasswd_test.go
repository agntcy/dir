// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package htpasswd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewManager(t *testing.T) {
	manager := NewManager("/test/path/htpasswd")
	if manager.GetPath() != "/test/path/htpasswd" {
		t.Errorf("Expected path '/test/path/htpasswd', got '%s'", manager.GetPath())
	}
}

func TestGenerateCredentials(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	htpasswdPath := filepath.Join(tempDir, "htpasswd")

	manager := NewManager(htpasswdPath)

	// Generate credentials
	creds, err := manager.GenerateCredentials("test-node-1")
	if err != nil {
		t.Fatalf("Failed to generate credentials: %v", err)
	}

	// Verify credentials structure
	expectedUsername := "sync-test-node-1"
	if creds.Username != expectedUsername {
		t.Errorf("Expected username '%s', got '%s'", expectedUsername, creds.Username)
	}

	if len(creds.Password) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Expected password length 32, got %d", len(creds.Password))
	}

	// Verify file was created
	if _, err := os.Stat(htpasswdPath); os.IsNotExist(err) {
		t.Errorf("htpasswd file was not created")
	}

	// Verify file contents
	content, err := os.ReadFile(htpasswdPath)
	if err != nil {
		t.Fatalf("Failed to read htpasswd file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line in htpasswd file, got %d", len(lines))
	}

	parts := strings.SplitN(lines[0], ":", 2)
	if len(parts) != 2 {
		t.Errorf("Invalid htpasswd format: %s", lines[0])
	}

	if parts[0] != expectedUsername {
		t.Errorf("Username in file '%s' doesn't match expected '%s'", parts[0], expectedUsername)
	}

	// Verify password hash
	err = bcrypt.CompareHashAndPassword([]byte(parts[1]), []byte(creds.Password))
	if err != nil {
		t.Errorf("Password hash verification failed: %v", err)
	}
}

func TestAddUser(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	htpasswdPath := filepath.Join(tempDir, "htpasswd")

	manager := NewManager(htpasswdPath)

	// Add a user
	err := manager.AddUser("testuser", "testpassword")
	if err != nil {
		t.Fatalf("Failed to add user: %v", err)
	}

	// Verify file contents
	content, err := os.ReadFile(htpasswdPath)
	if err != nil {
		t.Fatalf("Failed to read htpasswd file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line in htpasswd file, got %d", len(lines))
	}

	parts := strings.SplitN(lines[0], ":", 2)
	if parts[0] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", parts[0])
	}

	// Verify password hash
	err = bcrypt.CompareHashAndPassword([]byte(parts[1]), []byte("testpassword"))
	if err != nil {
		t.Errorf("Password hash verification failed: %v", err)
	}
}

func TestRemoveUser(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	htpasswdPath := filepath.Join(tempDir, "htpasswd")

	manager := NewManager(htpasswdPath)

	// Add two users
	err := manager.AddUser("user1", "password1")
	if err != nil {
		t.Fatalf("Failed to add user1: %v", err)
	}

	err = manager.AddUser("user2", "password2")
	if err != nil {
		t.Fatalf("Failed to add user2: %v", err)
	}

	// Verify both users exist
	users, err := manager.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// Remove one user
	err = manager.RemoveUser("user1")
	if err != nil {
		t.Fatalf("Failed to remove user: %v", err)
	}

	// Verify only one user remains
	users, err = manager.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users after removal: %v", err)
	}

	if len(users) != 1 {
		t.Errorf("Expected 1 user after removal, got %d", len(users))
	}

	if users[0] != "user2" {
		t.Errorf("Expected remaining user 'user2', got '%s'", users[0])
	}
}

func TestListUsers(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	htpasswdPath := filepath.Join(tempDir, "htpasswd")

	manager := NewManager(htpasswdPath)

	// List users on empty file
	users, err := manager.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users on empty file: %v", err)
	}

	if len(users) != 0 {
		t.Errorf("Expected 0 users on empty file, got %d", len(users))
	}

	// Add some users
	err = manager.AddUser("alice", "password1")
	if err != nil {
		t.Fatalf("Failed to add alice: %v", err)
	}

	err = manager.AddUser("bob", "password2")
	if err != nil {
		t.Fatalf("Failed to add bob: %v", err)
	}

	// List users
	users, err = manager.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}

	// Verify users are present (order may vary)
	userMap := make(map[string]bool)
	for _, user := range users {
		userMap[user] = true
	}

	if !userMap["alice"] {
		t.Errorf("User 'alice' not found in list")
	}

	if !userMap["bob"] {
		t.Errorf("User 'bob' not found in list")
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	htpasswdPath := filepath.Join(tempDir, "htpasswd")

	manager := NewManager(htpasswdPath)

	// Test concurrent access
	done := make(chan bool, 10)

	// Start multiple goroutines adding users
	for i := range 10 {
		go func(id int) {
			defer func() { done <- true }()

			err := manager.AddUser(
				fmt.Sprintf("user%d", id),
				fmt.Sprintf("password%d", id),
			)
			if err != nil {
				t.Errorf("Failed to add user%d: %v", id, err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Verify all users were added
	users, err := manager.ListUsers()
	if err != nil {
		t.Fatalf("Failed to list users after concurrent access: %v", err)
	}

	if len(users) != 10 {
		t.Errorf("Expected 10 users after concurrent access, got %d", len(users))
	}
}
