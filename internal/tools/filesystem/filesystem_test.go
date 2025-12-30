package filesystem

import (
	"encoding/json"
	"io/fs"
	"os"
	"testing"
	"time"
)

// MockFileSystem is a mock implementation of the FileSystem interface.
type MockFileSystem struct {
	StatFunc      func(name string) (os.FileInfo, error)
	ReadFileFunc  func(name string) ([]byte, error)
	WriteFileFunc func(name string, data []byte, perm os.FileMode) error
	MkdirAllFunc  func(path string, perm os.FileMode) error
	RemoveFunc    func(name string) error
	ReadDirFunc   func(name string) ([]os.DirEntry, error)
	WalkDirFunc   func(root string, fn fs.WalkDirFunc) error
}

func (m *MockFileSystem) Stat(name string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) ReadFile(name string) ([]byte, error) {
	if m.ReadFileFunc != nil {
		return m.ReadFileFunc(name)
	}
	return nil, os.ErrNotExist
}

func (m *MockFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	if m.WriteFileFunc != nil {
		return m.WriteFileFunc(name, data, perm)
	}
	return nil
}

func (m *MockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	if m.MkdirAllFunc != nil {
		return m.MkdirAllFunc(path, perm)
	}
	return nil
}

func (m *MockFileSystem) Remove(name string) error {
	if m.RemoveFunc != nil {
		return m.RemoveFunc(name)
	}
	return nil
}

func (m *MockFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	if m.ReadDirFunc != nil {
		return m.ReadDirFunc(name)
	}
	return nil, nil
}

func (m *MockFileSystem) WalkDir(root string, fn fs.WalkDirFunc) error {
	if m.WalkDirFunc != nil {
		return m.WalkDirFunc(root, fn)
	}
	return nil
}

// mockFileInfo implements os.FileInfo
type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() os.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Sys() any           { return nil }

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() os.FileMode          { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestReadFileImpl(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		mockContent string
		mockErr     error
		wantErr     bool
	}{
		{
			name:        "Read existing file",
			path:        "test.txt",
			mockContent: "hello world",
			wantErr:     false,
		},
		{
			name:    "Read non-existent file",
			path:    "missing.txt",
			mockErr: os.ErrNotExist,
			wantErr: true,
		},
		{
			name:    "Path traversal attempt",
			path:    "../secret.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &MockFileSystem{
				ReadFileFunc: func(name string) ([]byte, error) {
					return []byte(tt.mockContent), tt.mockErr
				},
			}

			_, err := readFileImpl(fs, "/repo", tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("readFileImpl() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWriteFileImpl(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		content string
		wantErr bool
	}{
		{
			name:    "Write file",
			path:    "test.txt",
			content: "hello",
			wantErr: false,
		},
		{
			name:    "Path traversal attempt",
			path:    "../test.txt",
			content: "hello",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &MockFileSystem{
				MkdirAllFunc: func(path string, perm os.FileMode) error { return nil },
				WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
					return nil
				},
			}

			_, err := writeFileImpl(fs, "/repo", tt.path, tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("writeFileImpl() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteFileImpl(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		mockStat os.FileInfo
		mockErr  error
		wantErr  bool
		wantMsg  string
	}{
		{
			name:     "Delete existing file",
			path:     "test.txt",
			mockStat: mockFileInfo{name: "test.txt", isDir: false},
			wantErr:  false,
		},
		{
			name:    "Delete non-existent file",
			path:    "missing.txt",
			mockErr: os.ErrNotExist,
			wantErr: false, // Should succeed with message
		},
		{
			name:     "Delete directory",
			path:     "dir",
			mockStat: mockFileInfo{name: "dir", isDir: true},
			wantErr:  true,
		},
		{
			name:    "Path traversal attempt",
			path:    "../test.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := &MockFileSystem{
				StatFunc: func(name string) (os.FileInfo, error) {
					return tt.mockStat, tt.mockErr
				},
				RemoveFunc: func(name string) error { return nil },
			}

			_, err := deleteFileImpl(fs, "/repo", tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("deleteFileImpl() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListFilesImpl(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		recursive      bool
		maxDepth       int
		limit          int
		ignorePatterns []string
		mockEntries    []os.DirEntry
		mockWalk       func(root string, fn fs.WalkDirFunc) error
		wantFiles      []string
		wantTruncated  bool
		wantErr        bool
	}{
		{
			name:      "List root non-recursive",
			path:      "",
			recursive: false,
			limit:     1000,
			mockEntries: []os.DirEntry{
				mockDirEntry{name: "file1.txt", isDir: false},
				mockDirEntry{name: "dir1", isDir: true},
				mockDirEntry{name: ".hidden", isDir: false},
			},
			wantFiles: []string{"file1.txt", "dir1"},
		},
		{
			name:           "List recursive with ignore",
			path:           "",
			recursive:      true,
			maxDepth:       -1,
			limit:          1000,
			ignorePatterns: []string{"*.log"},
			mockWalk: func(root string, fn fs.WalkDirFunc) error {
				// Simulate walk
				fn("/repo", mockDirEntry{name: "repo", isDir: true}, nil)
				fn("/repo/file1.txt", mockDirEntry{name: "file1.txt", isDir: false}, nil)
				fn("/repo/error.log", mockDirEntry{name: "error.log", isDir: false}, nil)
				fn("/repo/src", mockDirEntry{name: "src", isDir: true}, nil)
				fn("/repo/src/main.go", mockDirEntry{name: "main.go", isDir: false}, nil)
				return nil
			},
			wantFiles: []string{"file1.txt", "src", "src/main.go"},
		},
		{
			name:      "List recursive with max depth",
			path:      "",
			recursive: true,
			maxDepth:  0, // Only root children
			limit:     1000,
			mockWalk: func(root string, fn fs.WalkDirFunc) error {
				fn("/repo", mockDirEntry{name: "repo", isDir: true}, nil)
				fn("/repo/file1.txt", mockDirEntry{name: "file1.txt", isDir: false}, nil)
				fn("/repo/src", mockDirEntry{name: "src", isDir: true}, nil)
				fn("/repo/src/main.go", mockDirEntry{name: "main.go", isDir: false}, nil)
				return nil
			},
			wantFiles: []string{"file1.txt", "src"}, // src/main.go should be skipped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFS := &MockFileSystem{
				ReadDirFunc: func(name string) ([]os.DirEntry, error) {
					return tt.mockEntries, nil
				},
				WalkDirFunc: tt.mockWalk,
			}

			resultJSON, err := listFilesImpl(mockFS, "/repo", tt.path, tt.recursive, tt.maxDepth, tt.limit, tt.ignorePatterns)
			if (err != nil) != tt.wantErr {
				t.Errorf("listFilesImpl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				var result map[string]interface{}
				json.Unmarshal([]byte(resultJSON), &result)

				filesInterface := result["files"].([]interface{})
				files := make([]string, len(filesInterface))
				for i, v := range filesInterface {
					files[i] = v.(string)
				}

				if len(files) != len(tt.wantFiles) {
					t.Errorf("got %d files, want %d", len(files), len(tt.wantFiles))
				}

				// Simple check for presence
				for _, want := range tt.wantFiles {
					found := false
					for _, got := range files {
						if got == want {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("missing file %s", want)
					}
				}
			}
		})
	}
}
