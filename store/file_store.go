package store

import (
	"os"
	"path/filepath"
)

type FileMatchOptimizerStore struct {
	path string
}

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
