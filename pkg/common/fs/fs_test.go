package fs

import (
	"os"
	"path/filepath"
	"testing"

	"go4pack/pkg/common/compress"
)

func TestNew(t *testing.T) {
	tempDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)

	fsys, err := New()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if fsys == nil {
		t.Fatal("Expected filesystem instance, got nil")
	}

	// Check if runtime directories were created
	if _, err := os.Stat(".runtime"); os.IsNotExist(err) {
		t.Error("Expected .runtime directory to be created")
	}

	if _, err := os.Stat(".runtime/objects"); os.IsNotExist(err) {
		t.Error("Expected .runtime/objects directory to be created")
	}

	// Check paths
	expectedRuntimePath := filepath.Join(".", ".runtime")
	expectedObjectsPath := filepath.Join(".", ".runtime", "objects")

	if fsys.GetRuntimePath() != expectedRuntimePath {
		t.Errorf("Expected runtime path %s, got %s", expectedRuntimePath, fsys.GetRuntimePath())
	}

	if fsys.GetObjectsPath() != expectedObjectsPath {
		t.Errorf("Expected objects path %s, got %s", expectedObjectsPath, fsys.GetObjectsPath())
	}
}

func TestNewWithBasePath(t *testing.T) {
	tempDir := t.TempDir()
	customBase := filepath.Join(tempDir, "custom")

	fsys, err := NewWithBasePath(customBase)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expectedRuntimePath := filepath.Join(customBase, ".runtime")
	expectedObjectsPath := filepath.Join(customBase, ".runtime", "objects")

	if fsys.GetRuntimePath() != expectedRuntimePath {
		t.Errorf("Expected runtime path %s, got %s", expectedRuntimePath, fsys.GetRuntimePath())
	}

	if fsys.GetObjectsPath() != expectedObjectsPath {
		t.Errorf("Expected objects path %s, got %s", expectedObjectsPath, fsys.GetObjectsPath())
	}

	// Check if directories were created
	if _, err := os.Stat(expectedRuntimePath); os.IsNotExist(err) {
		t.Error("Expected runtime directory to be created")
	}

	if _, err := os.Stat(expectedObjectsPath); os.IsNotExist(err) {
		t.Error("Expected objects directory to be created")
	}
}

func TestGetFs(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	fs := fsys.GetFs()
	if fs == nil {
		t.Error("Expected Afero filesystem instance, got nil")
	}

	// Test that we can use it as an Afero filesystem
	_, err = fs.Stat(fsys.GetObjectsPath())
	if err != nil {
		t.Errorf("Failed to stat objects path with returned filesystem: %v", err)
	}
}

func TestWriteAndReadObject(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	testData := []byte("Hello, filesystem testing!")
	filename := "test.txt"

	// Test WriteObject
	err = fsys.WriteObject(filename, testData)
	if err != nil {
		t.Fatalf("Failed to write object: %v", err)
	}

	// Test ReadObject
	readData, err := fsys.ReadObject(filename)
	if err != nil {
		t.Fatalf("Failed to read object: %v", err)
	}

	if string(readData) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(readData))
	}

	// Test reading non-existent file
	_, err = fsys.ReadObject("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when reading non-existent file")
	}
}

func TestDeleteObject(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	filename := "delete_test.txt"
	testData := []byte("delete me")

	// Create object
	err = fsys.WriteObject(filename, testData)
	if err != nil {
		t.Fatalf("Failed to write object: %v", err)
	}

	// Verify it exists
	exists, err := fsys.ObjectExists(filename)
	if err != nil {
		t.Fatalf("Failed to check object existence: %v", err)
	}
	if !exists {
		t.Error("Expected object to exist before deletion")
	}

	// Delete object
	err = fsys.DeleteObject(filename)
	if err != nil {
		t.Fatalf("Failed to delete object: %v", err)
	}

	// Verify it's gone
	exists, err = fsys.ObjectExists(filename)
	if err != nil {
		t.Fatalf("Failed to check object existence after deletion: %v", err)
	}
	if exists {
		t.Error("Expected object to not exist after deletion")
	}

	// Test deleting non-existent file
	err = fsys.DeleteObject("nonexistent.txt")
	if err == nil {
		t.Error("Expected error when deleting non-existent file")
	}
}

func TestObjectExists(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	filename := "exists_test.txt"

	// Test non-existent file
	exists, err := fsys.ObjectExists(filename)
	if err != nil {
		t.Fatalf("Failed to check object existence: %v", err)
	}
	if exists {
		t.Error("Expected object to not exist")
	}

	// Create file
	err = fsys.WriteObject(filename, []byte("test"))
	if err != nil {
		t.Fatalf("Failed to write object: %v", err)
	}

	// Test existing file
	exists, err = fsys.ObjectExists(filename)
	if err != nil {
		t.Fatalf("Failed to check object existence: %v", err)
	}
	if !exists {
		t.Error("Expected object to exist")
	}
}

func TestListObjects(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	// Test empty directory
	objects, err := fsys.ListObjects()
	if err != nil {
		t.Fatalf("Failed to list objects: %v", err)
	}
	if len(objects) != 0 {
		t.Errorf("Expected 0 objects, got %d", len(objects))
	}

	// Create some objects
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range testFiles {
		err = fsys.WriteObject(filename, []byte("test content"))
		if err != nil {
			t.Fatalf("Failed to write object %s: %v", filename, err)
		}
	}

	// List objects
	objects, err = fsys.ListObjects()
	if err != nil {
		t.Fatalf("Failed to list objects: %v", err)
	}

	if len(objects) != len(testFiles) {
		t.Errorf("Expected %d objects, got %d", len(testFiles), len(objects))
	}

	// Check all files are present
	objectMap := make(map[string]bool)
	for _, obj := range objects {
		objectMap[obj] = true
	}

	for _, expectedFile := range testFiles {
		if !objectMap[expectedFile] {
			t.Errorf("Expected file %s not found in objects list", expectedFile)
		}
	}
}

func TestGetObjectInfo(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	filename := "info_test.txt"
	testData := []byte("test data for info")

	// Create object
	err = fsys.WriteObject(filename, testData)
	if err != nil {
		t.Fatalf("Failed to write object: %v", err)
	}

	// Get object info
	info, err := fsys.GetObjectInfo(filename)
	if err != nil {
		t.Fatalf("Failed to get object info: %v", err)
	}

	if info.Name() != filename {
		t.Errorf("Expected name %s, got %s", filename, info.Name())
	}

	// Note: Size will be the compressed size, which may be different from original
	if info.Size() <= 0 {
		t.Errorf("Expected positive size, got %d", info.Size())
	}

	if info.IsDir() {
		t.Error("Expected file, not directory")
	}

	// Test that we can get the original size too
	originalSize, err := fsys.GetOriginalObjectSize(filename)
	if err != nil {
		t.Fatalf("Failed to get original object size: %v", err)
	}

	if originalSize != int64(len(testData)) {
		t.Errorf("Expected original size %d, got %d", len(testData), originalSize)
	}

	// Test non-existent file
	_, err = fsys.GetObjectInfo("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestGetObjectSize(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	filename := "size_test.txt"
	testData := []byte("test data for size calculation")

	// Create object
	err = fsys.WriteObject(filename, testData)
	if err != nil {
		t.Fatalf("Failed to write object: %v", err)
	}

	// Get compressed object size
	compressedSize, err := fsys.GetObjectSize(filename)
	if err != nil {
		t.Fatalf("Failed to get object size: %v", err)
	}

	if compressedSize <= 0 {
		t.Errorf("Expected positive compressed size, got %d", compressedSize)
	}

	// Get original object size
	originalSize, err := fsys.GetOriginalObjectSize(filename)
	if err != nil {
		t.Fatalf("Failed to get original object size: %v", err)
	}

	expectedSize := int64(len(testData))
	if originalSize != expectedSize {
		t.Errorf("Expected original size %d, got %d", expectedSize, originalSize)
	}

	// Test non-existent file
	_, err = fsys.GetObjectSize("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	_, err = fsys.GetOriginalObjectSize("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestCopyObjectTo(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	sourceFile := "source.txt"
	destFile := filepath.Join(tempDir, "destination.txt")
	testData := []byte("copy test data")

	// Create source object
	err = fsys.WriteObject(sourceFile, testData)
	if err != nil {
		t.Fatalf("Failed to write source object: %v", err)
	}

	// Copy object
	err = fsys.CopyObjectTo(sourceFile, destFile)
	if err != nil {
		t.Fatalf("Failed to copy object: %v", err)
	}

	// Verify destination file exists and has correct content
	copiedData, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedData) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(copiedData))
	}

	// Test copying non-existent file
	err = fsys.CopyObjectTo("nonexistent.txt", filepath.Join(tempDir, "dest2.txt"))
	if err == nil {
		t.Error("Expected error when copying non-existent file")
	}
}

func TestCreateObjectDir(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	dirname := "testdir"

	// Create directory
	err = fsys.CreateObjectDir(dirname)
	if err != nil {
		t.Fatalf("Failed to create object directory: %v", err)
	}

	// Verify directory exists
	dirPath := filepath.Join(fsys.GetObjectsPath(), dirname)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}

	// Test creating nested directory
	nestedDir := "level1/level2/level3"
	err = fsys.CreateObjectDir(nestedDir)
	if err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	nestedPath := filepath.Join(fsys.GetObjectsPath(), nestedDir)
	if _, err := os.Stat(nestedPath); os.IsNotExist(err) {
		t.Error("Expected nested directory to be created")
	}
}

func TestWriteAndReadObjectFromDir(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	dirname := "subdir"
	filename := "file.txt"
	testData := []byte("subdirectory test data")

	// Write to subdirectory (should create directory automatically)
	err = fsys.WriteObjectToDir(dirname, filename, testData)
	if err != nil {
		t.Fatalf("Failed to write object to directory: %v", err)
	}

	// Verify directory was created
	dirPath := filepath.Join(fsys.GetObjectsPath(), dirname)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		t.Error("Expected directory to be created")
	}

	// Read from subdirectory
	readData, err := fsys.ReadObjectFromDir(dirname, filename)
	if err != nil {
		t.Fatalf("Failed to read object from directory: %v", err)
	}

	if string(readData) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(readData))
	}

	// Test reading non-existent file from directory
	_, err = fsys.ReadObjectFromDir(dirname, "nonexistent.txt")
	if err == nil {
		t.Error("Expected error when reading non-existent file from directory")
	}

	// Test reading from non-existent directory
	_, err = fsys.ReadObjectFromDir("nonexistent", filename)
	if err == nil {
		t.Error("Expected error when reading from non-existent directory")
	}
}

func TestCleanObjects(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	// Create some objects and directories
	err = fsys.WriteObject("file1.txt", []byte("test1"))
	if err != nil {
		t.Fatalf("Failed to write file1: %v", err)
	}

	err = fsys.WriteObject("file2.txt", []byte("test2"))
	if err != nil {
		t.Fatalf("Failed to write file2: %v", err)
	}

	err = fsys.WriteObjectToDir("subdir", "file3.txt", []byte("test3"))
	if err != nil {
		t.Fatalf("Failed to write file3 to subdir: %v", err)
	}

	// Verify objects exist
	objects, err := fsys.ListObjects()
	if err != nil {
		t.Fatalf("Failed to list objects before clean: %v", err)
	}
	if len(objects) != 2 { // file1.txt and file2.txt (subdir is not counted as an object)
		t.Errorf("Expected 2 objects before clean, got %d", len(objects))
	}

	// Clean objects
	err = fsys.CleanObjects()
	if err != nil {
		t.Fatalf("Failed to clean objects: %v", err)
	}

	// Verify objects are gone
	objects, err = fsys.ListObjects()
	if err != nil {
		t.Fatalf("Failed to list objects after clean: %v", err)
	}
	if len(objects) != 0 {
		t.Errorf("Expected 0 objects after clean, got %d", len(objects))
	}

	// Verify subdirectory is also gone
	subdirPath := filepath.Join(fsys.GetObjectsPath(), "subdir")
	if _, err := os.Stat(subdirPath); !os.IsNotExist(err) {
		t.Error("Expected subdirectory to be removed")
	}
}

// Benchmark tests
func BenchmarkWriteObject(b *testing.B) {
	tempDir := b.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		b.Fatalf("Failed to create filesystem: %v", err)
	}

	testData := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := filepath.Join("bench", "file", "path", "test.txt")
		fsys.WriteObject(filename, testData)
	}
}

func BenchmarkReadObject(b *testing.B) {
	tempDir := b.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		b.Fatalf("Failed to create filesystem: %v", err)
	}

	testData := []byte("benchmark test data")
	filename := "benchmark.txt"

	// Create test file
	err = fsys.WriteObject(filename, testData)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsys.ReadObject(filename)
	}
}

func BenchmarkObjectExists(b *testing.B) {
	tempDir := b.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		b.Fatalf("Failed to create filesystem: %v", err)
	}

	filename := "benchmark.txt"
	err = fsys.WriteObject(filename, []byte("test"))
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsys.ObjectExists(filename)
	}
}

func BenchmarkListObjects(b *testing.B) {
	tempDir := b.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		b.Fatalf("Failed to create filesystem: %v", err)
	}

	// Create multiple test files
	for i := 0; i < 100; i++ {
		filename := filepath.Join("file", "path", "test") + string(rune(i)) + ".txt"
		fsys.WriteObject(filename, []byte("test data"))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fsys.ListObjects()
	}
}

// Compression-related tests
func TestNewWithCompression(t *testing.T) {
	tempDir := t.TempDir()
	comp := compress.NewZstdCompressorMax()

	fsys, err := NewWithBasePathAndCompression(tempDir, comp)
	if err != nil {
		t.Fatalf("Failed to create filesystem with compression: %v", err)
	}

	if fsys.GetCompressor().Type() != compress.Zstd {
		t.Errorf("Expected Zstd compressor, got %v", fsys.GetCompressor().Type())
	}
}

func TestCompressionInWriteAndRead(t *testing.T) {
	tempDir := t.TempDir()

	// Use zstd compression
	zstdComp := compress.NewDefaultCompressor()
	fsys, err := NewWithBasePathAndCompression(tempDir, zstdComp)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	testData := []byte("This is test data that should be compressed using zstd compression algorithm.")
	filename := "compressed_test.txt"

	// Write object
	if err = fsys.WriteObject(filename, testData); err != nil {
		t.Fatalf("Failed to write compressed object: %v", err)
	}

	// Read object
	readData, err := fsys.ReadObject(filename)
	if err != nil {
		t.Fatalf("Failed to read compressed object: %v", err)
	}
	if string(readData) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(readData))
	}
}

func TestSetCompressor(t *testing.T) {
	tempDir := t.TempDir()
	fsys, err := NewWithBasePath(tempDir)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	// Default should be Zstd now
	if fsys.GetCompressor().Type() != compress.Zstd {
		t.Errorf("Expected default compressor to be Zstd, got %v", fsys.GetCompressor().Type())
	}

	// Change to no compression
	none := compress.NewNoneCompressor()
	fsys.SetCompressor(none)
	if fsys.GetCompressor().Type() != compress.None {
		t.Errorf("Expected None compressor after setting, got %v", fsys.GetCompressor().Type())
	}
}

func TestCompressionWithDirectories(t *testing.T) {
	tempDir := t.TempDir()
	zstdComp := compress.NewDefaultCompressor()
	fsys, err := NewWithBasePathAndCompression(tempDir, zstdComp)
	if err != nil {
		t.Fatalf("Failed to create filesystem: %v", err)
	}

	dirname := "compressed_subdir"
	filename := "compressed_file.txt"
	testData := []byte("This is test data in a subdirectory that should be compressed (zstd).")

	if err = fsys.WriteObjectToDir(dirname, filename, testData); err != nil {
		t.Fatalf("Failed to write compressed object to directory: %v", err)
	}
	readData, err := fsys.ReadObjectFromDir(dirname, filename)
	if err != nil {
		t.Fatalf("Failed to read compressed object from directory: %v", err)
	}
	if string(readData) != string(testData) {
		t.Errorf("Expected %s, got %s", string(testData), string(readData))
	}
}
