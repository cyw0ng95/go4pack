package file

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"

	"github.com/gabriel-vasile/mimetype"
)

// MD5Sum returns the lowercase hex MD5 checksum of the provided data.
func MD5Sum(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

// DetectMIME attempts to determine the MIME type from content first, then falls back to filename / standard detection.
// Returns mime type string like "text/plain" or "application/octet-stream" on failure.
func DetectMIME(data []byte, filename string) string {
	if len(data) > 0 {
		if m := mimetype.Detect(data); m != nil {
			return m.String()
		}
	}
	// Fallback to net/http sniffing (uses at most first 512 bytes)
	if len(data) > 0 {
		return http.DetectContentType(data)
	}
	return "application/octet-stream"
}
