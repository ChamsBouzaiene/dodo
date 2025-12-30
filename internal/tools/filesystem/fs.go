package filesystem

import (
	"io/fs"
	"os"
	"path/filepath"
)

// FileSystem defines the interface for filesystem operations.
// This allows mocking the os package for testing.
type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
	Remove(name string) error
	ReadDir(name string) ([]os.DirEntry, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
}

// OSFileSystem is the default implementation that uses the os package.
type OSFileSystem struct{}

// NewOSFileSystem creates a new OSFileSystem.
func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (fs *OSFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (fs *OSFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (fs *OSFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *OSFileSystem) Remove(name string) error {
	return os.Remove(name)
}

func (fs *OSFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}

func (fs *OSFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, fn)
}
