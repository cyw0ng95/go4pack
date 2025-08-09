package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// CompressionType represents the type of compression algorithm
type CompressionType int

const (
	// None represents no compression
	None CompressionType = iota
	// Gzip represents gzip compression
	Gzip
)

// String returns the string representation of the compression type
func (ct CompressionType) String() string {
	switch ct {
	case None:
		return "none"
	case Gzip:
		return "gzip"
	default:
		return "unknown"
	}
}

// Compressor interface defines methods for data compression
type Compressor interface {
	// Compress compresses the input data and returns compressed data
	Compress(data []byte) ([]byte, error)
	// Decompress decompresses the input data and returns original data
	Decompress(data []byte) ([]byte, error)
	// Type returns the compression type
	Type() CompressionType
}

// gzipCompressor implements Compressor interface using gzip
type gzipCompressor struct {
	level int
}

// NewGzipCompressor creates a new gzip compressor with specified compression level
// level should be between 1 (fastest) and 9 (best compression), or gzip.DefaultCompression
func NewGzipCompressor(level int) Compressor {
	return &gzipCompressor{level: level}
}

// Compress compresses data using gzip
func (gc *gzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gc.level)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("failed to write data to gzip writer: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress decompresses gzip data
func (gc *gzipCompressor) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read from gzip reader: %w", err)
	}

	return result, nil
}

// Type returns the compression type
func (gc *gzipCompressor) Type() CompressionType {
	return Gzip
}

// noneCompressor implements Compressor interface with no compression
type noneCompressor struct{}

// NewNoneCompressor creates a new compressor that performs no compression
func NewNoneCompressor() Compressor {
	return &noneCompressor{}
}

// Compress returns the data as-is (no compression)
func (nc *noneCompressor) Compress(data []byte) ([]byte, error) {
	return data, nil
}

// Decompress returns the data as-is (no decompression)
func (nc *noneCompressor) Decompress(data []byte) ([]byte, error) {
	return data, nil
}

// Type returns the compression type
func (nc *noneCompressor) Type() CompressionType {
	return None
}

// NewCompressor creates a new compressor based on the specified type
func NewCompressor(cType CompressionType) Compressor {
	switch cType {
	case Gzip:
		return NewGzipCompressor(gzip.DefaultCompression)
	case None:
		return NewNoneCompressor()
	default:
		return NewNoneCompressor()
	}
}

// NewDefaultCompressor creates a new compressor with default settings (gzip with default compression)
func NewDefaultCompressor() Compressor {
	return NewGzipCompressor(gzip.DefaultCompression)
}

// CompressWithType compresses data using the specified compression type
func CompressWithType(data []byte, cType CompressionType) ([]byte, error) {
	compressor := NewCompressor(cType)
	return compressor.Compress(data)
}

// DecompressWithType decompresses data using the specified compression type
func DecompressWithType(data []byte, cType CompressionType) ([]byte, error) {
	compressor := NewCompressor(cType)
	return compressor.Decompress(data)
}

// IsCompressed checks if the data appears to be compressed by examining magic bytes
func IsCompressed(data []byte) CompressionType {
	if len(data) < 2 {
		return None
	}

	// Check for gzip magic number (0x1f, 0x8b)
	if data[0] == 0x1f && data[1] == 0x8b {
		return Gzip
	}

	return None
}
