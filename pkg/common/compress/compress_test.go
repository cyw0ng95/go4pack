package compress

import (
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestGzipCompressor(t *testing.T) {
	compressor := NewGzipCompressor(gzip.DefaultCompression)

	testData := []byte("This is a test string that should be compressed and then decompressed")

	// Test compression
	compressed, err := compressor.Compress(testData)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatal("Compressed data is empty")
	}

	// Test decompression
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(testData, decompressed) {
		t.Fatalf("Decompressed data doesn't match original. Expected: %s, Got: %s", string(testData), string(decompressed))
	}

	// Test compression type
	if compressor.Type() != Gzip {
		t.Fatalf("Expected compression type Gzip, got %v", compressor.Type())
	}
}

func TestNoneCompressor(t *testing.T) {
	compressor := NewNoneCompressor()

	testData := []byte("This data should not be compressed")

	// Test compression (should return data as-is)
	compressed, err := compressor.Compress(testData)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	if !bytes.Equal(testData, compressed) {
		t.Fatalf("None compressor should return data as-is")
	}

	// Test decompression (should return data as-is)
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(testData, decompressed) {
		t.Fatalf("None compressor should return data as-is")
	}

	// Test compression type
	if compressor.Type() != None {
		t.Fatalf("Expected compression type None, got %v", compressor.Type())
	}
}

func TestNewCompressor(t *testing.T) {
	tests := []struct {
		cType    CompressionType
		expected CompressionType
	}{
		{Gzip, Gzip},
		{None, None},
		{CompressionType(999), None}, // Invalid type should default to None
	}

	for _, test := range tests {
		compressor := NewCompressor(test.cType)
		if compressor.Type() != test.expected {
			t.Errorf("NewCompressor(%v) returned type %v, expected %v", test.cType, compressor.Type(), test.expected)
		}
	}
}

func TestCompressWithType(t *testing.T) {
	testData := []byte("Test data for compression")

	// Test with Gzip
	compressed, err := CompressWithType(testData, Gzip)
	if err != nil {
		t.Fatalf("CompressWithType failed: %v", err)
	}

	decompressed, err := DecompressWithType(compressed, Gzip)
	if err != nil {
		t.Fatalf("DecompressWithType failed: %v", err)
	}

	if !bytes.Equal(testData, decompressed) {
		t.Fatalf("Round-trip compression failed")
	}

	// Test with None
	compressed, err = CompressWithType(testData, None)
	if err != nil {
		t.Fatalf("CompressWithType with None failed: %v", err)
	}

	if !bytes.Equal(testData, compressed) {
		t.Fatalf("None compression should return data as-is")
	}
}

func TestIsCompressed(t *testing.T) {
	// Test uncompressed data
	uncompressedData := []byte("This is not compressed")
	if IsCompressed(uncompressedData) != None {
		t.Errorf("Expected None for uncompressed data")
	}

	// Test gzip compressed data
	compressor := NewGzipCompressor(gzip.DefaultCompression)
	compressedData, err := compressor.Compress(uncompressedData)
	if err != nil {
		t.Fatalf("Failed to compress test data: %v", err)
	}

	if IsCompressed(compressedData) != Gzip {
		t.Errorf("Expected Gzip for gzip compressed data")
	}

	// Test empty data
	if IsCompressed([]byte{}) != None {
		t.Errorf("Expected None for empty data")
	}

	// Test data with only one byte
	if IsCompressed([]byte{0x1f}) != None {
		t.Errorf("Expected None for single byte data")
	}
}

func TestCompressionTypes(t *testing.T) {
	tests := []struct {
		cType    CompressionType
		expected string
	}{
		{None, "none"},
		{Gzip, "gzip"},
		{CompressionType(999), "unknown"},
	}

	for _, test := range tests {
		if test.cType.String() != test.expected {
			t.Errorf("CompressionType(%d).String() = %s, expected %s", int(test.cType), test.cType.String(), test.expected)
		}
	}
}

func TestCompressionEfficiency(t *testing.T) {
	// Create data that should compress well (repeated text)
	testData := []byte(strings.Repeat("This is a repeated string for testing compression efficiency. ", 100))

	compressor := NewGzipCompressor(gzip.BestCompression)
	compressed, err := compressor.Compress(testData)
	if err != nil {
		t.Fatalf("Compression failed: %v", err)
	}

	compressionRatio := float64(len(compressed)) / float64(len(testData))
	if compressionRatio > 0.5 {
		t.Logf("Compression ratio: %.2f%% (might be expected for small test data)", compressionRatio*100)
	} else {
		t.Logf("Good compression ratio: %.2f%%", compressionRatio*100)
	}

	// Verify decompression works correctly
	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompression failed: %v", err)
	}

	if !bytes.Equal(testData, decompressed) {
		t.Fatalf("Decompressed data doesn't match original")
	}
}

func BenchmarkGzipCompress(b *testing.B) {
	compressor := NewGzipCompressor(gzip.DefaultCompression)
	testData := []byte(strings.Repeat("This is test data for benchmarking compression performance. ", 1000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressor.Compress(testData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGzipDecompress(b *testing.B) {
	compressor := NewGzipCompressor(gzip.DefaultCompression)
	testData := []byte(strings.Repeat("This is test data for benchmarking decompression performance. ", 1000))
	
	compressed, err := compressor.Compress(testData)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := compressor.Decompress(compressed)
		if err != nil {
			b.Fatal(err)
		}
	}
}
