// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sessionstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	fileUtils "github.com/agntcy/dir/hub/utils/file"
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

func (s *FileSecretStore) SaveHubSession(sessionKey string, session *HubSession) error {
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

	var sessions HubSessions
	if err = json.NewDecoder(file).Decode(&sessions); err != nil {
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: %w", ErrMalformedSecret, err)
		}
	}

	if sessions.HubSessions == nil {
		sessions.HubSessions = make(map[string]*HubSession)
	}

	sessions.HubSessions[sessionKey] = session

	if err = rewriteJSONFilePretty(file, sessions); err != nil {
		return fmt.Errorf("%w: %w", ErrCouldNotWriteFile, err)
	}

	return nil
}

func (s *FileSecretStore) RemoveSession(sessionKey string) error {
	sessions, file, err := s.getSessionsAndFile()
	if err != nil {
		return err
	}

	if file == nil || sessions == nil {
		return nil
	}

	defer file.Close()

	delete(sessions.HubSessions, sessionKey)

	if err = rewriteJSONFilePretty(file, sessions); err != nil {
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
