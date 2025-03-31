package file

import (
	"os"
	"path/filepath"

	"github.com/agntcy/dir/cli/util/dir"
)

func CreateAll(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, err
	}

	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func GetSecretsFilePath() string {
	return filepath.Join(dir.GetAppDir(), "secrets.json")
}
