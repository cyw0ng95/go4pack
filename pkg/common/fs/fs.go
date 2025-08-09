package fs

import (
	"fmt"
	"os"
	"path/filepath"

	"go4pack/pkg/common/compress"

	"github.com/spf13/afero"
)

// FileSystem wraps Afero filesystem with runtime object management
type FileSystem struct {
	fs          afero.Fs
	runtimePath string
	objectsPath string
	compressor  compress.Compressor
}

// New creates a new filesystem instance with runtime directory management
func New() (*FileSystem, error) {
	return NewWithBasePath(".")
}

// NewWithBasePath creates a new filesystem instance with custom base path
func NewWithBasePath(basePath string) (*FileSystem, error) {
	return NewWithBasePathAndCompression(basePath, compress.NewDefaultCompressor())
}

// NewWithCompression creates a new filesystem instance with custom compression
func NewWithCompression(compressor compress.Compressor) (*FileSystem, error) {
	return NewWithBasePathAndCompression(".", compressor)
}

// NewWithBasePathAndCompression creates a new filesystem instance with custom base path and compression
func NewWithBasePathAndCompression(basePath string, compressor compress.Compressor) (*FileSystem, error) {
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
		compressor:  compressor,
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

// GetCompressor returns the current compressor
func (fsys *FileSystem) GetCompressor() compress.Compressor {
	return fsys.compressor
}

// SetCompressor sets a new compressor for the filesystem
func (fsys *FileSystem) SetCompressor(compressor compress.Compressor) {
	fsys.compressor = compressor
}

// WriteObject writes data to a file in the objects directory with compression unless data already compressed.
func (fsys *FileSystem) WriteObject(filename string, data []byte) error {
	// Avoid double compression: if detectable format, store raw.
	if ct := compress.IsCompressed(data); ct != compress.None {
		objectPath := filepath.Join(fsys.objectsPath, filename)
		return afero.WriteFile(fsys.fs, objectPath, data, 0644)
	}
	compressedData, err := fsys.compressor.Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}
	objectPath := filepath.Join(fsys.objectsPath, filename)
	return afero.WriteFile(fsys.fs, objectPath, compressedData, 0644)
}

// WriteObjectWithMIME writes data, skipping compression if already compressed per magic or MIME.
func (fsys *FileSystem) WriteObjectWithMIME(filename string, data []byte, mime string) error {
	if ct := compress.IsCompressedOrMIME(data, mime); ct != compress.None {
		objectPath := filepath.Join(fsys.objectsPath, filename)
		return afero.WriteFile(fsys.fs, objectPath, data, 0644)
	}
	return fsys.WriteObject(filename, data)
}

// ReadObject reads data from a file in the objects directory with decompression
func (fsys *FileSystem) ReadObject(filename string) ([]byte, error) {
	objectPath := filepath.Join(fsys.objectsPath, filename)
	compressedData, err := afero.ReadFile(fsys.fs, objectPath)
	if err != nil {
		return nil, err
	}
	// detect explicit formats first
	detectedType := compress.IsCompressed(compressedData)
	if detectedType != compress.None {
		return compress.DecompressWithType(compressedData, detectedType)
	}
	// fallback (may already be uncompressed)
	return fsys.safeDecompress(compressedData)
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

// CopyObjectTo copies an object to another location (decompressed)
func (fsys *FileSystem) CopyObjectTo(filename, destPath string) error {
	// Read the object data (this will decompress it)
	data, err := fsys.ReadObject(filename)
	if err != nil {
		return fmt.Errorf("failed to read source object: %w", err)
	}

	destFile, err := fsys.fs.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = destFile.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to destination file: %w", err)
	}

	return nil
}

// CreateObjectDir creates a subdirectory in the objects directory
func (fsys *FileSystem) CreateObjectDir(dirname string) error {
	dirPath := filepath.Join(fsys.objectsPath, dirname)
	return fsys.fs.MkdirAll(dirPath, 0755)
}

// WriteObjectToDir writes data to a file in a subdirectory of objects with compression unless already compressed.
func (fsys *FileSystem) WriteObjectToDir(dirname, filename string, data []byte) error {
	dirPath := filepath.Join(fsys.objectsPath, dirname)
	if err := fsys.fs.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create object directory: %w", err)
	}
	objectPath := filepath.Join(dirPath, filename)
	if ct := compress.IsCompressed(data); ct != compress.None {
		return afero.WriteFile(fsys.fs, objectPath, data, 0644)
	}
	compressedData, err := fsys.compressor.Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}
	return afero.WriteFile(fsys.fs, objectPath, compressedData, 0644)
}

// ReadObjectFromDir reads data from a file in a subdirectory of objects with decompression
func (fsys *FileSystem) ReadObjectFromDir(dirname, filename string) ([]byte, error) {
	objectPath := filepath.Join(fsys.objectsPath, dirname, filename)
	compressedData, err := afero.ReadFile(fsys.fs, objectPath)
	if err != nil {
		return nil, err
	}
	detectedType := compress.IsCompressed(compressedData)
	if detectedType != compress.None {
		return compress.DecompressWithType(compressedData, detectedType)
	}
	return fsys.safeDecompress(compressedData)
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

// GetObjectSize returns the size of an object file (compressed size on disk)
func (fsys *FileSystem) GetObjectSize(filename string) (int64, error) {
	info, err := fsys.GetObjectInfo(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// GetOriginalObjectSize returns the uncompressed size of an object
func (fsys *FileSystem) GetOriginalObjectSize(filename string) (int64, error) {
	data, err := fsys.ReadObject(filename)
	if err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

// hashedPath returns the storage path for a given hex hash (expects length >=2)
func (fsys *FileSystem) hashedPath(hash string) string {
	if len(hash) < 2 {
		return filepath.Join(fsys.objectsPath, hash) // fallback
	}
	return filepath.Join(fsys.objectsPath, hash[:2], hash)
}

// WriteObjectHashed stores data under a path derived from its hash with compression unless data already compressed.
// If the file already exists, it is left untouched.
func (fsys *FileSystem) WriteObjectHashed(hash string, data []byte) error {
	p := fsys.hashedPath(hash)
	dir := filepath.Dir(p)
	if err := fsys.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create hash directory: %w", err)
	}
	// If exists, skip re-writing (deduplicate)
	if exists, _ := afero.Exists(fsys.fs, p); exists {
		return nil
	}
	if ct := compress.IsCompressed(data); ct != compress.None {
		return afero.WriteFile(fsys.fs, p, data, 0644)
	}
	compressedData, err := fsys.compressor.Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress data: %w", err)
	}
	return afero.WriteFile(fsys.fs, p, compressedData, 0644)
}

// WriteObjectHashedWithMIME hashed write with MIME-aware double compression avoidance.
func (fsys *FileSystem) WriteObjectHashedWithMIME(hash string, data []byte, mime string) error {
	if ct := compress.IsCompressedOrMIME(data, mime); ct != compress.None {
		return fsys.WriteObjectHashedRaw(hash, data)
	}
	return fsys.WriteObjectHashed(hash, data)
}

// WriteObjectHashedRaw stores data under its hash without applying additional compression (dedup aware).
func (fsys *FileSystem) WriteObjectHashedRaw(hash string, data []byte) error {
	p := fsys.hashedPath(hash)
	dir := filepath.Dir(p)
	if err := fsys.fs.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create hash directory: %w", err)
	}
	if exists, _ := afero.Exists(fsys.fs, p); exists {
		return nil
	}
	return afero.WriteFile(fsys.fs, p, data, 0644)
}

// safeDecompress tries to decompress with current compressor; on failure returns original data.
func (fsys *FileSystem) safeDecompress(data []byte) ([]byte, error) {
	out, err := fsys.compressor.Decompress(data)
	if err != nil {
		// treat as uncompressed original
		return data, nil
	}
	return out, nil
}

// ReadObjectHashed reads a hashed (content-addressed) object.
func (fsys *FileSystem) ReadObjectHashed(hash string) ([]byte, error) {
	p := fsys.hashedPath(hash)
	compressedData, err := afero.ReadFile(fsys.fs, p)
	if err != nil {
		return nil, err
	}
	detectedType := compress.IsCompressed(compressedData)
	if detectedType != compress.None {
		return compress.DecompressWithType(compressedData, detectedType)
	}
	return fsys.safeDecompress(compressedData)
}

// GetHashedObjectSize returns compressed size of hashed object.
func (fsys *FileSystem) GetHashedObjectSize(hash string) (int64, error) {
	p := fsys.hashedPath(hash)
	info, err := fsys.fs.Stat(p)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// HashedObjectPath returns the filesystem path where a given hash would be stored.
func (fsys *FileSystem) HashedObjectPath(hash string) string { return fsys.hashedPath(hash) }

// CommitTempAsHashed moves a temp file into its hashed location unless an object already exists.
// Returns final path and a boolean indicating whether new file was stored.
func (fsys *FileSystem) CommitTempAsHashed(tempFilePath, hash string) (string, bool, error) {
	p := fsys.hashedPath(hash)
	dir := filepath.Dir(p)
	if err := fsys.fs.MkdirAll(dir, 0755); err != nil {
		return "", false, fmt.Errorf("create hash dir: %w", err)
	}
	if exists, _ := afero.Exists(fsys.fs, p); exists {
		// discard temp (dedup)
		_ = fsys.fs.Remove(tempFilePath)
		return p, false, nil
	}
	// rename (must use os.Rename for real FS; afero OsFs implements)
	if _, ok := fsys.fs.(*afero.OsFs); ok {
		if err := os.Rename(tempFilePath, p); err != nil {
			return "", false, fmt.Errorf("rename temp: %w", err)
		}
	} else {
		// fallback: copy then remove
		data, err := afero.ReadFile(fsys.fs, tempFilePath)
		if err != nil {
			return "", false, fmt.Errorf("read temp: %w", err)
		}
		if err := afero.WriteFile(fsys.fs, p, data, 0644); err != nil {
			return "", false, fmt.Errorf("write hashed: %w", err)
		}
		_ = fsys.fs.Remove(tempFilePath)
	}
	return p, true, nil
}
