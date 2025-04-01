package secretstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	fileUtils "github.com/agntcy/dir/cli/util/file"
)

type FileSecretStore struct {
	path string
}

func NewFileSecretStore(path string) *FileSecretStore {
	return &FileSecretStore{path: path}
}

func (s *FileSecretStore) GetHubSecret(secretName string) (*HubSecret, error) {
	secrets, err := s.getSecrets()
	if err != nil {
		return nil, err
	}

	secret, ok := secrets.HubSecrets[secretName]
	if !ok || secret == nil {
		return nil, fmt.Errorf("%w: %s", ErrSecretNotFound, secretName)
	}

	if err = secret.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidSecret, err)
	}

	return secret, nil
}

func (s *FileSecretStore) SaveHubSecret(secretName string, secret *HubSecret) error {
	if err := secret.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidSecret, err)
	}

	file, err := os.OpenFile(s.path, os.O_RDWR|os.O_CREATE, 0600)
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

	var secrets HubSecrets
	if err = json.NewDecoder(file).Decode(&secrets); err != nil {
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("%w: %w", ErrMalformedSecret, err)
		}
	}

	if secrets.HubSecrets == nil {
		secrets.HubSecrets = make(map[string]*HubSecret)
	}
	secrets.HubSecrets[secretName] = secret

	file.Seek(0, 0)
	file.Truncate(0)
	if err = json.NewEncoder(file).Encode(&secrets); err != nil {
		return fmt.Errorf("%w: %w", ErrCouldNotWriteFile, err)
	}

	return nil
}

func (s *FileSecretStore) RemoveHubSecret(secretName string) error {
	secrets, file, err := s.getSecretsAndFile()
	if err != nil {
		return err
	}
	if file == nil {
		return nil
	}
	defer file.Close()

	if _, ok := secrets.HubSecrets[secretName]; !ok {
		return nil
	}

	delete(secrets.HubSecrets, secretName)

	file.Seek(0, 0)
	file.Truncate(0)
	if err = json.NewEncoder(file).Encode(&secrets); err != nil {
		return fmt.Errorf("%w: %w", ErrCouldNotWriteFile, err)
	}

	return nil
}

func (s *FileSecretStore) getSecretsAndFile() (*HubSecrets, *os.File, error) {
	file, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &HubSecrets{}, nil, nil
		}
		return nil, nil, fmt.Errorf("%w: %w: %s", ErrCouldNotOpenFile, err, s.path)
	}

	var secrets *HubSecrets
	if err = json.NewDecoder(file).Decode(&secrets); err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("%w: %w", ErrMalformedSecretFile, err)
	}

	return secrets, file, nil
}

func (s *FileSecretStore) getSecrets() (*HubSecrets, error) {
	secrets, file, err := s.getSecretsAndFile()
	defer file.Close()
	return secrets, err
}
