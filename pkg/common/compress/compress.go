package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
)

// CompressionType represents the type of compression algorithm
type CompressionType int

const (
	// None represents no compression
	None CompressionType = iota
	// Gzip represents gzip compression
	Gzip
	// Zstd represents zstandard compression
	Zstd
)

// String returns the string representation of the compression type
func (ct CompressionType) String() string {
	switch ct {
	case None:
		return "none"
	case Gzip:
		return "gzip"
	case Zstd:
		return "zstd"
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

// zstdCompressor implements Compressor interface using zstandard
type zstdCompressor struct {
	encoderLevel zstd.EncoderLevel
}

// NewZstdCompressorMax creates a new zstd compressor with maximum compression
func NewZstdCompressorMax() Compressor {
	return &zstdCompressor{encoderLevel: zstd.SpeedBestCompression}
}

// NewZstdCompressor creates a new zstd compressor with specified compression level
func NewZstdCompressor(level zstd.EncoderLevel) Compressor {
	return &zstdCompressor{encoderLevel: level}
}

// Compress compresses data using zstandard
func (zc *zstdCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	enc, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zc.encoderLevel))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd encoder: %w", err)
	}

	if _, err := enc.Write(data); err != nil {
		enc.Close()
		return nil, fmt.Errorf("failed to write data to zstd encoder: %w", err)
	}

	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zstd encoder: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress decompresses zstandard data
func (zc *zstdCompressor) Decompress(data []byte) ([]byte, error) {
	dec, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer dec.Close()

	result, err := io.ReadAll(dec)
	if err != nil {
		return nil, fmt.Errorf("failed to read from zstd decoder: %w", err)
	}

	return result, nil
}

// Type returns the compression type
func (zc *zstdCompressor) Type() CompressionType {
	return Zstd
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
	case Zstd:
		return NewZstdCompressorMax()
	case None:
		return NewNoneCompressor()
	default:
		return NewNoneCompressor()
	}
}

// NewDefaultCompressor creates a new compressor with default settings (zstd with max compression)
func NewDefaultCompressor() Compressor {
	return NewZstdCompressorMax()
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

	// Check for zstd frame magic (0x28, 0xB5, 0x2F, 0xFD)
	if len(data) >= 4 && data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD {
		return Zstd
	}

	return None
}

var mimeCompressionMap = map[string]CompressionType{
	"application/gzip":   Gzip,
	"application/x-gzip": Gzip,
	"application/zstd":   Zstd,
	"application/x-zstd": Zstd,
}

// DetectCompressionByMIME returns the compression type inferred from MIME if known.
func DetectCompressionByMIME(mime string) (CompressionType, bool) {
	ct, ok := mimeCompressionMap[mime]
	return ct, ok
}

// IsCompressedOrMIME first checks magic bytes, then MIME hint (if provided, empty mime ignored).
func IsCompressedOrMIME(data []byte, mime string) CompressionType {
	if ct := IsCompressed(data); ct != None {
		return ct
	}
	if mime == "" {
		return None
	}
	if ct, ok := DetectCompressionByMIME(mime); ok {
		return ct
	}
	return None
}
