package fileio

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go4pack/pkg/common/logger"
	"go4pack/pkg/common/worker"
)

// scheduleRpmAnalysis performs lightweight RPM header parsing (no payload extraction)
// and stores cached JSON metadata. Best-effort only; errors captured in JSON.
func scheduleRpmAnalysis(recID uint, data []byte) {
	_ = worker.Submit(func() {
		logger.GetLogger().Debug().Uint("record_id", recID).Msg("starting async RPM analysis")
		meta := map[string]any{"analyzed_at": time.Now().UTC().Format(time.RFC3339)}
		analysis, err := parseRPMHeaders(data)
		if err != nil {
			meta["error"] = err.Error()
		} else {
			for k, v := range analysis {
				meta[k] = v
			}
		}
		b, _ := json.Marshal(meta)

		db, dErr := ensureDB()
		if dErr != nil {
			return
		}
		cache := &RpmAnalyzeCached{FileID: recID, Data: string(b)}
		_ = db.Where("file_id = ?", recID).Assign(map[string]any{"data": cache.Data}).FirstOrCreate(cache)

		status := "done"
		if err != nil {
			status = "error"
		}
		db.Model(&FileRecord{}).Where("id = ?", recID).Update("analysis_status", status)
		logger.GetLogger().Info().Uint("record_id", recID).Str("status", status).Msg("rpm analysis completed")
	})
}

// parseRPMHeaders extracts basic metadata from an RPM file (name/version/release/arch/etc.)
// It parses the signature header and main header; payload is not processed.
func parseRPMHeaders(data []byte) (map[string]any, error) {
	const (
		leadSize = 96
		magic0   = 0x8e
		magic1   = 0xad
		magic2   = 0xe8
	)
	if len(data) < leadSize+16 {
		return nil, errors.New("file too small for rpm")
	}
	// basic lead sanity: bytes 0-1 should be 0xed ab? historical lead magic is not enforced; skip.
	// We rely on header magic.
	off := leadSize
	readHeader := func(start int) (nIndex, hSize int, next int, err error) {
		if start+16 > len(data) {
			return 0, 0, 0, errors.New("truncated header")
		}
		if data[start] != magic0 || data[start+1] != magic1 || data[start+2] != magic2 {
			return 0, 0, 0, errors.New("bad header magic")
		}
		nIndex = int(binary.BigEndian.Uint32(data[start+8 : start+12]))
		hSize = int(binary.BigEndian.Uint32(data[start+12 : start+16]))
		entryBytes := nIndex * 16
		storeStart := start + 16 + entryBytes
		storeEnd := storeStart + hSize
		if storeEnd > len(data) {
			return 0, 0, 0, errors.New("truncated store")
		}
		next = storeEnd
		// align to 8
		if rem := next % 8; rem != 0 {
			next += (8 - rem)
		}
		return
	}
	// signature header
	nIdx, hSz, next, err := readHeader(off)
	if err != nil {
		return nil, fmt.Errorf("signature header: %w", err)
	}
	off = next
	_ = nIdx
	_ = hSz // unused currently
	// main header
	nIdx, hSz, _, err = readHeader(off)
	if err != nil {
		return nil, fmt.Errorf("main header: %w", err)
	}

	entriesStart := off + 16
	storeStart := entriesStart + nIdx*16
	storeEnd := storeStart + hSz
	if storeEnd > len(data) {
		return nil, errors.New("truncated main store")
	}
	store := data[storeStart:storeEnd]

	getString := func(offset int) string {
		if offset < 0 || offset >= len(store) {
			return ""
		}
		end := offset
		for end < len(store) && store[end] != 0 {
			end++
		}
		return string(store[offset:end])
	}
	// Tag constants
	const (
		TAG_NAME              = 1000
		TAG_VERSION           = 1001
		TAG_RELEASE           = 1002
		TAG_SUMMARY           = 1004
		TAG_LICENSE           = 1014
		TAG_ARCH              = 1022
		TAG_PAYLOADFORMAT     = 1124
		TAG_PAYLOADCOMPRESSOR = 1125
		TAG_PAYLOADFLAGS      = 1126
	)
	// Type constants
	const (
		RPM_STRING     = 6
		RPM_I18NSTRING = 9
	)
	meta := map[string]any{}
	for i := 0; i < nIdx; i++ {
		entry := data[entriesStart+i*16 : entriesStart+(i+1)*16]
		tag := int(binary.BigEndian.Uint32(entry[0:4]))
		typeID := int(binary.BigEndian.Uint32(entry[4:8]))
		offset := int(binary.BigEndian.Uint32(entry[8:12]))
		count := int(binary.BigEndian.Uint32(entry[12:16]))
		if count <= 0 {
			continue
		}
		if typeID == RPM_STRING || typeID == RPM_I18NSTRING {
			val := getString(offset)
			switch tag {
			case TAG_NAME:
				meta["name"] = val
			case TAG_VERSION:
				meta["version"] = val
			case TAG_RELEASE:
				meta["release"] = val
			case TAG_SUMMARY:
				meta["summary"] = val
			case TAG_LICENSE:
				meta["license"] = val
			case TAG_ARCH:
				meta["arch"] = val
			case TAG_PAYLOADFORMAT:
				meta["payload_format"] = val
			case TAG_PAYLOADCOMPRESSOR:
				meta["payload_compressor"] = val
			case TAG_PAYLOADFLAGS:
				meta["payload_flags"] = val
			}
		}
	}
	return meta, nil
}
