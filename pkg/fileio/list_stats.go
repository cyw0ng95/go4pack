package fileio

import (
	"encoding/json"
	iofs "io/fs"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"

	elfutil "go4pack/pkg/common/elf"
	"go4pack/pkg/common/fs"
	"go4pack/pkg/common/logger"
)

func listHandler(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "50")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}

	db, err := ensureDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database init failed"})
		return
	}
	var total int64
	if err := db.Model(&FileRecord{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "count failed"})
		return
	}
	var files []FileRecord
	offset := (page - 1) * pageSize
	if err := db.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query files failed"})
		return
	}
	resp := make([]gin.H, 0, len(files))
	for _, f := range files {
		isELF := f.AnalysisStatus == "pending" || f.AnalysisStatus == "error"
		if f.AnalysisStatus == "done" {
			isELF = true
		}
		isGzip := (f.MIME == "application/gzip" || f.MIME == "application/x-gzip")
		avail := []string{}
		if isELF {
			avail = append(avail, "elf")
		}
		if isGzip {
			avail = append(avail, "gzip")
		}
		resp = append(resp, gin.H{
			"id":                 f.ID,
			"filename":           f.Filename,
			"size":               f.Size,
			"compressed_size":    f.CompressedSize,
			"compression_type":   f.CompressionType,
			"md5":                f.MD5,
			"mime":               f.MIME,
			"created_at":         f.CreatedAt,
			"updated_at":         f.UpdatedAt,
			"is_elf":             isELF,
			"is_gzip":            isGzip,
			"analysis_status":    f.AnalysisStatus,
			"available_analysis": avail, // NEW
		})
	}
	pages := (total + int64(pageSize) - 1) / int64(pageSize)
	logger.GetLogger().Info().Int("count", len(files)).Int64("total", total).Int("page", page).Int("page_size", pageSize).Msg("files listed paginated")
	c.JSON(http.StatusOK, gin.H{"files": resp, "count": len(files), "total": total, "page": page, "page_size": pageSize, "pages": pages})
}

func statsHandler(c *gin.Context) {
	db, err := ensureDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database init failed"})
		return
	}
	var files []FileRecord
	if err := db.Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query files failed"})
		return
	}
	var totalOriginalSize, totalCompressedSize int64
	compressionStats := make(map[string]int)
	mimeStats := make(map[string]int)
	uniqueHashSeen := make(map[string]struct{})
	var uniqueCompressedSize int64
	for _, file := range files {
		totalOriginalSize += file.Size
		totalCompressedSize += file.CompressedSize
		compressionStats[file.CompressionType]++
		mimeStats[file.MIME]++
		if _, ok := uniqueHashSeen[file.MD5]; !ok {
			uniqueHashSeen[file.MD5] = struct{}{}
			uniqueCompressedSize += file.CompressedSize
		}
	}
	var compressionRatio float64
	if totalOriginalSize > 0 {
		compressionRatio = float64(totalCompressedSize) / float64(totalOriginalSize)
	}
	spaceSaved := totalOriginalSize - totalCompressedSize
	var spaceSavedPct float64
	if totalOriginalSize > 0 {
		spaceSavedPct = float64(spaceSaved) / float64(totalOriginalSize) * 100
	}
	physicalObjectsCount := 0
	var physicalObjectsSize int64
	if fsys, err := fs.New(); err == nil {
		root := fsys.GetObjectsPath()
		_ = filepath.WalkDir(root, func(path string, d iofs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, e := d.Info()
			if e != nil {
				return nil
			}
			physicalObjectsCount++
			physicalObjectsSize += info.Size()
			return nil
		})
	}
	var dedupSavedCompressed int64 = totalCompressedSize - physicalObjectsSize
	if dedupSavedCompressed < 0 {
		dedupSavedCompressed = 0
	}
	var dedupSavedCompressedPct float64
	if totalCompressedSize > 0 {
		dedupSavedCompressedPct = float64(dedupSavedCompressed) / float64(totalCompressedSize) * 100
	}
	var dedupSavedOriginal int64 = totalOriginalSize - physicalObjectsSize
	if dedupSavedOriginal < 0 {
		dedupSavedOriginal = 0
	}
	var dedupSavedOriginalPct float64
	if totalOriginalSize > 0 {
		dedupSavedOriginalPct = float64(dedupSavedOriginal) / float64(totalOriginalSize) * 100
	}
	logger.GetLogger().Info().Int("file_count", len(files)).Int("unique_hash_count", len(uniqueHashSeen)).Int64("logical_original", totalOriginalSize).Int64("logical_compressed", totalCompressedSize).Int64("physical_compressed", physicalObjectsSize).Float64("compression_ratio", compressionRatio).Msg("compression & dedup stats requested")
	c.JSON(http.StatusOK, gin.H{"file_count": len(files), "unique_hash_count": len(uniqueHashSeen), "total_original_size": totalOriginalSize, "total_compressed_size": totalCompressedSize, "compression_ratio": compressionRatio, "space_saved": spaceSaved, "space_saved_percentage": spaceSavedPct, "compression_types": compressionStats, "mime_types": mimeStats, "unique_compressed_size": uniqueCompressedSize, "physical_objects_count": physicalObjectsCount, "physical_objects_size": physicalObjectsSize, "dedup_saved_compressed": dedupSavedCompressed, "dedup_saved_compr_pct": dedupSavedCompressedPct, "dedup_saved_original": dedupSavedOriginal, "dedup_saved_original_pct": dedupSavedOriginalPct})
}

func metaHandler(c *gin.Context) {
	idParam := c.Param("id")
	if idParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	db, err := ensureDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db init failed"})
		return
	}
	var fr FileRecord
	if err := db.First(&fr, idParam).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	reqType := c.Query("type") // "", "elf", "gzip"
	if reqType != "" && reqType != "elf" && reqType != "gzip" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type (expected elf|gzip)"})
		return
	}

	isGzip := fr.MIME == "application/gzip" || fr.MIME == "application/x-gzip"
	// We consider ELF if status not none (pending/done/error) or magic can be confirmed on demand
	isELFStatus := fr.AnalysisStatus == "pending" || fr.AnalysisStatus == "done" || fr.AnalysisStatus == "error"

	// Decide target analysis type
	var target string
	if reqType != "" {
		target = reqType
	} else {
		if isGzip {
			target = "gzip"
		} else if isELFStatus {
			target = "elf"
		}
	}

	// Validate compatibility if user requested a type mismatching file characteristics
	if reqType == "gzip" && !isGzip {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is not gzip"})
		return
	}
	if reqType == "elf" && !isELFStatus {
		// we can still probe magic to upgrade
		if fsys, ferr := fs.New(); ferr == nil {
			if data, rerr := fsys.ReadObjectHashed(fr.MD5); rerr == nil && len(data) >= 4 &&
				data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
				isELFStatus = true
			}
		}
		if !isELFStatus {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file is not ELF"})
			return
		}
	}

	resp := gin.H{"file": fr}

	// NEW: advertise available analyses
	avail := []string{}
	if fr.AnalysisStatus == "pending" || fr.AnalysisStatus == "done" || fr.AnalysisStatus == "error" {
		avail = append(avail, "elf")
	}
	if fr.MIME == "application/gzip" || fr.MIME == "application/x-gzip" {
		avail = append(avail, "gzip")
	}
	resp["available_analysis"] = avail

	switch target {
	case "elf":
		var cache ElfAnalyzeCached
		cacheFound := false
		if err := db.Where("file_id = ?", fr.ID).First(&cache).Error; err == nil {
			cacheFound = true
		} else {
			// On-demand compute if not error status
			if fr.AnalysisStatus != "error" {
				if fsys, ferr := fs.New(); ferr == nil {
					if data, rerr := fsys.ReadObjectHashed(fr.MD5); rerr == nil && len(data) >= 4 &&
						data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F' {
						if analysisMap, aerr := elfutil.AnalyzeBytes(data); aerr == nil {
							if b, mErr := json.Marshal(analysisMap); mErr == nil {
								cache = ElfAnalyzeCached{FileID: fr.ID, Data: string(b)}
								_ = db.Create(&cache).Error
								if fr.AnalysisStatus != "done" {
									_ = db.Model(&FileRecord{}).Where("id = ?", fr.ID).Update("analysis_status", "done").Error
									fr.AnalysisStatus = "done"
								}
								cacheFound = true
							}
						} else {
							msg := aerr.Error()
							_ = db.Model(&FileRecord{}).Where("id = ?", fr.ID).
								Updates(map[string]any{"analysis_status": "error", "analysis_error": msg})
							fr.AnalysisStatus = "error"
						}
					}
				}
			}
		}
		resp["analysis_type"] = "elf"
		if cacheFound {
			resp["analysis"] = json.RawMessage(cache.Data)
		} else {
			resp["analysis"] = nil
		}
	case "gzip":
		var gcache GzipAnalyzeCached
		if err := db.Where("file_id = ?", fr.ID).First(&gcache).Error; err == nil {
			resp["analysis_type"] = "gzip"
			resp["analysis"] = json.RawMessage(gcache.Data)
		} else {
			resp["analysis_type"] = "gzip"
			resp["analysis"] = nil
		}
	default:
		// No analysis requested/detected
		resp["analysis_type"] = nil
		resp["analysis"] = nil
	}

	resp["analysis_status"] = fr.AnalysisStatus
	c.JSON(http.StatusOK, resp)
}

// Provide JSON raw marshal reuse (kept for consistency with former file)
var _ = json.RawMessage{}
