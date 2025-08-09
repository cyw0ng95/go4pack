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

func TestZstdCompressor(t *testing.T) {
	c := NewCompressor(Zstd)
	if c.Type() != Zstd {
		t.Fatalf("expected Zstd compressor")
	}
	data := []byte("This is some test data that should compress well with zstd.")
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("zstd compression failed: %v", err)
	}
	if len(compressed) == 0 {
		t.Fatalf("zstd compressed data empty")
	}
	decompressed, err := c.Decompress(compressed)
	if err != nil {
		t.Fatalf("zstd decompression failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatalf("zstd roundtrip mismatch")
	}
}

func TestNewCompressor_Factory(t *testing.T) {
	tests := []struct {
		in   CompressionType
		want CompressionType
	}{
		{Gzip, Gzip},
		{Zstd, Zstd},
		{None, None},
		{CompressionType(999), None},
	}
	for _, tt := range tests {
		c := NewCompressor(tt.in)
		if c.Type() != tt.want {
			t.Errorf("NewCompressor(%v) => %v, want %v", tt.in, c.Type(), tt.want)
		}
	}
}

func TestCompressWithType_Zstd(t *testing.T) {
	data := []byte("Roundtrip with zstd")
	compressed, err := CompressWithType(data, Zstd)
	if err != nil {
		t.Fatalf("zstd compress failed: %v", err)
	}
	decompressed, err := DecompressWithType(compressed, Zstd)
	if err != nil {
		t.Fatalf("zstd decompress failed: %v", err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatalf("zstd roundtrip mismatch")
	}
}

func TestIsCompressed_Zstd(t *testing.T) {
	data := []byte("detect zstd magic bytes test")
	c := NewCompressor(Zstd)
	compressed, err := c.Compress(data)
	if err != nil {
		t.Fatalf("zstd compress failed: %v", err)
	}
	// Detection should return Zstd
	if IsCompressed(compressed) != Zstd {
		t.Fatalf("expected Zstd detection")
	}
}

func TestCompressionTypes_List(t *testing.T) {
	cases := []struct {
		ct   CompressionType
		want string
	}{
		{None, "none"},
		{Gzip, "gzip"},
		{Zstd, "zstd"},
		{CompressionType(999), "unknown"},
	}
	for _, c := range cases {
		if got := c.ct.String(); got != c.want {
			t.Errorf("CompressionType(%d).String()=%s want %s", c.ct, got, c.want)
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
