package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// FileSystem wraps Afero filesystem with runtime object management
type FileSystem struct {
	fs          afero.Fs
	runtimePath string
	objectsPath string
}

// New creates a new filesystem instance with runtime directory management
func New() (*FileSystem, error) {
	return NewWithBasePath(".")
}

// NewWithBasePath creates a new filesystem instance with custom base path
func NewWithBasePath(basePath string) (*FileSystem, error) {
	fs := afero.NewOsFs()
	runtimePath := filepath.Join(basePath, ".runtime")
	objectsPath := filepath.Join(runtimePath, "objects")

	// Create runtime directories if they don't exist
	if err := fs.MkdirAll(objectsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create runtime directories: %w", err)
	}

	return &FileSystem{
		fs:          fs,
		runtimePath: runtimePath,
		objectsPath: objectsPath,
	}, nil
}

// GetFs returns the underlying Afero filesystem
func (fsys *FileSystem) GetFs() afero.Fs {
	return fsys.fs
}

// GetRuntimePath returns the .runtime directory path
func (fsys *FileSystem) GetRuntimePath() string {
	return fsys.runtimePath
}

// GetObjectsPath returns the .runtime/objects directory path
func (fsys *FileSystem) GetObjectsPath() string {
	return fsys.objectsPath
}

// WriteObject writes data to a file in the objects directory
func (fsys *FileSystem) WriteObject(filename string, data []byte) error {
	objectPath := filepath.Join(fsys.objectsPath, filename)
	return afero.WriteFile(fsys.fs, objectPath, data, 0644)
}

// ReadObject reads data from a file in the objects directory
func (fsys *FileSystem) ReadObject(filename string) ([]byte, error) {
	objectPath := filepath.Join(fsys.objectsPath, filename)
	return afero.ReadFile(fsys.fs, objectPath)
}

// DeleteObject deletes a file from the objects directory
func (fsys *FileSystem) DeleteObject(filename string) error {
	objectPath := filepath.Join(fsys.objectsPath, filename)
	return fsys.fs.Remove(objectPath)
}

// ListObjects lists all files in the objects directory
func (fsys *FileSystem) ListObjects() ([]string, error) {
	entries, err := afero.ReadDir(fsys.fs, fsys.objectsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// ObjectExists checks if an object file exists
func (fsys *FileSystem) ObjectExists(filename string) (bool, error) {
	objectPath := filepath.Join(fsys.objectsPath, filename)
	return afero.Exists(fsys.fs, objectPath)
}

// GetObjectInfo returns file info for an object
func (fsys *FileSystem) GetObjectInfo(filename string) (os.FileInfo, error) {
	objectPath := filepath.Join(fsys.objectsPath, filename)
	return fsys.fs.Stat(objectPath)
}

// CopyObjectTo copies an object to another location
func (fsys *FileSystem) CopyObjectTo(filename, destPath string) error {
	objectPath := filepath.Join(fsys.objectsPath, filename)

	srcFile, err := fsys.fs.Open(objectPath)
	if err != nil {
		return fmt.Errorf("failed to open source object: %w", err)
	}
	defer srcFile.Close()

	destFile, err := fsys.fs.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return nil
}

// CreateObjectDir creates a subdirectory in the objects directory
func (fsys *FileSystem) CreateObjectDir(dirname string) error {
	dirPath := filepath.Join(fsys.objectsPath, dirname)
	return fsys.fs.MkdirAll(dirPath, 0755)
}

// WriteObjectToDir writes data to a file in a subdirectory of objects
func (fsys *FileSystem) WriteObjectToDir(dirname, filename string, data []byte) error {
	dirPath := filepath.Join(fsys.objectsPath, dirname)

	// Ensure directory exists
	if err := fsys.fs.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create object directory: %w", err)
	}

	objectPath := filepath.Join(dirPath, filename)
	return afero.WriteFile(fsys.fs, objectPath, data, 0644)
}

// ReadObjectFromDir reads data from a file in a subdirectory of objects
func (fsys *FileSystem) ReadObjectFromDir(dirname, filename string) ([]byte, error) {
	objectPath := filepath.Join(fsys.objectsPath, dirname, filename)
	return afero.ReadFile(fsys.fs, objectPath)
}

// CleanObjects removes all files from the objects directory
func (fsys *FileSystem) CleanObjects() error {
	entries, err := afero.ReadDir(fsys.fs, fsys.objectsPath)
	if err != nil {
		return fmt.Errorf("failed to read objects directory: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(fsys.objectsPath, entry.Name())
		if err := fsys.fs.RemoveAll(entryPath); err != nil {
			return fmt.Errorf("failed to remove %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// GetObjectSize returns the size of an object file
func (fsys *FileSystem) GetObjectSize(filename string) (int64, error) {
	info, err := fsys.GetObjectInfo(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
