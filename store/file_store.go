package store

import (
	"os"
	"path/filepath"
)

// FileMatchOptimizerStore persists match-optimizer state to a single JSON file.
// It reproduces the exact on-disk behaviour the optimizer used before the store
// abstraction was introduced.
type FileMatchOptimizerStore struct {
	path string
}

// NewFileMatchOptimizerStore returns a store backed by the file at path.
func NewFileMatchOptimizerStore(path string) *FileMatchOptimizerStore {
	return &FileMatchOptimizerStore{path: path}
}

func (s *FileMatchOptimizerStore) Load() ([]byte, error) {
	return os.ReadFile(s.path)
}

func (s *FileMatchOptimizerStore) Save(data []byte) error {
	dir := filepath.Dir(s.path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	file, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	return err
}
