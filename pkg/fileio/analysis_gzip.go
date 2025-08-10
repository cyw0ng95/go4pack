package fileio

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"time"

	"go4pack/pkg/common/worker"
)

// scheduleGzipAnalysis submits async job to analyze gzip (streaming to temp to avoid OOM)
func scheduleGzipAnalysis(recID uint, raw []byte) {
	_ = worker.Submit(func() {
		db, err := ensureDB()
		if err != nil {
			return
		}
		meta := map[string]any{
			"analyzed_at": time.Now().UTC().Format(time.RFC3339),
		}

		tempDir := ".runtime/temp"
		_ = os.MkdirAll(tempDir, 0o755)
		tmpPath := tempDir + "/gzip-" + strconv.FormatUint(uint64(recID), 10) + ".gz"
		if err := os.WriteFile(tmpPath, raw, 0o644); err != nil {
			meta["error"] = "write temp failed: " + err.Error()
			b, _ := json.Marshal(meta)
			cache := &GzipAnalyzeCached{FileID: recID, Data: string(b)}
			_ = db.Where("file_id = ?", recID).
				Assign(map[string]any{"data": cache.Data}).FirstOrCreate(cache)
			return
		}
		defer os.Remove(tmpPath)

		f, err := os.Open(tmpPath)
		if err != nil {
			meta["error"] = err.Error()
			b, _ := json.Marshal(meta)
			cache := &GzipAnalyzeCached{FileID: recID, Data: string(b)}
			_ = db.Where("file_id = ?", recID).
				Assign(map[string]any{"data": cache.Data}).FirstOrCreate(cache)
			return
		}
		defer f.Close()

		gr, err := gzip.NewReader(f)
		if err != nil {
			meta["error"] = err.Error()
			b, _ := json.Marshal(meta)
			cache := &GzipAnalyzeCached{FileID: recID, Data: string(b)}
			_ = db.Where("file_id = ?", recID).
				Assign(map[string]any{"data": cache.Data}).FirstOrCreate(cache)
			return
		}

		tr := tar.NewReader(gr)
		const maxEntries = 200
		var (
			entries          []map[string]any
			uncompressedSize int64
			isTar            = true
		)

		for isTar {
			h, e := tr.Next()
			if e == io.EOF {
				break
			}
			if e != nil {
				isTar = false
				break
			}
			entries = append(entries, map[string]any{
				"name": h.Name,
				"size": h.Size,
				"mode": h.Mode,
				"type": h.Typeflag,
			})
			if h.Size > 0 {
				n, _ := io.CopyN(io.Discard, tr, h.Size)
				uncompressedSize += n
			}
			if len(entries) >= maxEntries {
				meta["truncated"] = true
				break
			}
		}

		if !isTar && len(entries) == 0 {
			gr.Close()
			_, _ = f.Seek(0, 0)
			gr2, g2 := gzip.NewReader(f)
			if g2 != nil {
				meta["error"] = g2.Error()
			} else {
				n, _ := io.Copy(io.Discard, gr2)
				uncompressedSize = n
				gr2.Close()
			}
		} else {
			if isTar {
				nTail, _ := io.Copy(io.Discard, tr)
				uncompressedSize += nTail
			}
			gr.Close()
		}

		if uncompressedSize > 0 {
			meta["uncompressed_size"] = uncompressedSize
		}
		if len(entries) > 0 {
			meta["tar_entries"] = entries
			meta["tar_count"] = len(entries)
		}

		b, _ := json.Marshal(meta)
		cache := &GzipAnalyzeCached{FileID: recID, Data: string(b)}
		_ = db.Where("file_id = ?", recID).
			Assign(map[string]any{"data": cache.Data}).FirstOrCreate(cache)

		status := "done"
		if _, hasErr := meta["error"]; hasErr {
			status = "error"
		}
		db.Model(&FileRecord{}).Where("id = ?", recID).Update("analysis_status", status)
	})
}
