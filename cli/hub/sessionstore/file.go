// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sessionstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	fileUtils "github.com/agntcy/dir/cli/util/file"
)

const (
	ModeCurrentUserReadWrite os.FileMode = 0o600
)

type FileSecretStore struct {
	path string
}

func NewFileSessionStore(path string) *FileSecretStore {
	return &FileSecretStore{path: path}
}

func (s *FileSecretStore) GetHubSession(sessionKey string) (*HubSession, error) {
	secrets, err := s.getSessions()
	if err != nil {
		return nil, err
	}

	secret, ok := secrets.HubSessions[sessionKey]
	if !ok || secret == nil {
		return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, sessionKey)
	}

	return secret, nil
}

func (s *FileSecretStore) SaveHubSession(secretName string, secret *HubSession) error {
	file, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, ModeCurrentUserReadWrite)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			var err error
			if file, err = fileUtils.CreateAll(s.path); err != nil {
				return fmt.Errorf("%w: %w", ErrCouldNotOpenFile, err)
			}
		} else {
			return fmt.Errorf("%w: %w: %s", ErrCouldNotOpenFile, err, s.path)
		}
	}
	defer file.Close()

	var secrets HubSessions
	if err = json.NewDecoder(file).Decode(&secrets); err != nil {
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: %w", ErrMalformedSecret, err)
		}
	}

	if secrets.HubSessions == nil {
		secrets.HubSessions = make(map[string]*HubSession)
	}

	secrets.HubSessions[secretName] = secret

	if err = rewriteJSONFilePretty(file, secrets); err != nil {
		return fmt.Errorf("%w: %w", ErrCouldNotWriteFile, err)
	}

	return nil
}

func (s *FileSecretStore) RemoveHubSession(secretName string) error {
	secrets, file, err := s.getSessionsAndFile()
	if err != nil {
		return err
	}

	if file == nil {
		return nil
	}

	defer file.Close()

	if _, ok := secrets.HubSessions[secretName]; !ok {
		return nil
	}

	delete(secrets.HubSessions, secretName)

	if err = rewriteJSONFilePretty(file, secrets); err != nil {
		return fmt.Errorf("%w: %w", ErrCouldNotWriteFile, err)
	}

	return nil
}

func (s *FileSecretStore) getSessionsAndFile() (*HubSessions, *os.File, error) {
	file, err := os.OpenFile(s.path, os.O_RDWR, ModeCurrentUserReadWrite)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &HubSessions{}, nil, nil
		}

		return nil, nil, fmt.Errorf("%w: %w: %s", ErrCouldNotOpenFile, err, s.path)
	}

	var secrets *HubSessions
	if err = json.NewDecoder(file).Decode(&secrets); err != nil {
		file.Close()

		return nil, nil, fmt.Errorf("%w: %w", ErrMalformedSecretFile, err)
	}

	return secrets, file, nil
}

func (s *FileSecretStore) getSessions() (*HubSessions, error) {
	secrets, file, err := s.getSessionsAndFile()
	//nolint:errcheck
	defer file.Close()

	return secrets, err
}

func rewriteJSONFilePretty(file *os.File, model any) error {
	if file == nil {
		return errors.New("file is nil")
	}

	//nolint:errcheck
	file.Seek(0, 0)
	//nolint:errcheck
	file.Truncate(0)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(model); err != nil {
		return fmt.Errorf("%w: %w", ErrCouldNotWriteFile, err)
	}

	return nil
}
