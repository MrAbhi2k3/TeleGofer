package session

import (
	"os"
	"path/filepath"
)

// File persists session data to disk. The parent directory is
// created on save. File permissions are restricted to 0600.
type File struct {
	path string
}

func NewFile(path string) *File {
	return &File{path: path}
}

func (f *File) Load() ([]byte, error) {
	data, err := os.ReadFile(f.path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	return data, err
}

func (f *File) Save(data []byte) error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0700); err != nil {
		return err
	}
	return os.WriteFile(f.path, data, 0600)
}
