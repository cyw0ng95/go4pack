package fileio

import (
	"io"
	iofs "io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go4pack/pkg/common/compress"
	"go4pack/pkg/common/database"
	"go4pack/pkg/common/file"
	"go4pack/pkg/common/fs"
	"go4pack/pkg/common/logger"
)

// FileRecord represents a stored file metadata entry
type FileRecord struct {
	ID              uint           `gorm:"primaryKey" json:"id"`
	Filename        string         `gorm:"uniqueIndex;size:255" json:"filename"`
	Size            int64          `json:"size"`             // Original uncompressed size
	CompressedSize  int64          `json:"compressed_size"`  // Compressed size on disk
	CompressionType string         `json:"compression_type"` // Type of compression used
	MD5             string         `json:"md5"`
	MIME            string         `json:"mime"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// ensureDB migrates and returns db
func ensureDB() (*gorm.DB, error) {
	if db := database.Get(); db != nil {
		return db, nil
	}
	return database.Init("filemeta.db", &FileRecord{})
}

// RegisterRoutes registers file upload/download routes under given router group
func RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/upload", uploadHandler)
	rg.GET("/download/:filename", downloadHandler)
	rg.GET("/list", listHandler)
	rg.GET("/stats", statsHandler)
	rg.POST("/upload/multi", uploadMultiHandler)
}

func uploadHandler(c *gin.Context) {
	fileHdr, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer fileHdr.Close()

	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}

	data, err := io.ReadAll(fileHdr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read file failed"})
		return
	}

	originalSize := int64(len(data))
	md5sum := file.MD5Sum(data)
	mimeType := file.DetectMIME(data, header.Filename)
	preCT := compress.IsCompressedOrMIME(data, mimeType)

	// Content-addressed storage by MD5 (first 2 chars directory) with MIME-based compression avoidance
	if err := fsys.WriteObjectHashedWithMIME(md5sum, data, mimeType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store file failed"})
		return
	}

	compressedSize, err := fsys.GetHashedObjectSize(md5sum)
	if err != nil {
		logger.GetLogger().Warn().Err(err).Str("hash", md5sum).Msg("failed to get compressed size")
		compressedSize = originalSize
	}
	compressionType := fsys.GetCompressor().Type().String()
	if preCT != compress.None { // override if already compressed before storing
		compressionType = preCT.String()
	}

	if db, err := ensureDB(); err == nil {
		rec := &FileRecord{
			Filename:        header.Filename,
			Size:            originalSize,
			CompressedSize:  compressedSize,
			CompressionType: compressionType,
			MD5:             md5sum,
			MIME:            mimeType,
		}
		if err := db.Create(rec).Error; err != nil {
			logger.GetLogger().Error().Err(err).Str("filename", header.Filename).Msg("db create file record failed")
		}
	} else {
		logger.GetLogger().Error().Err(err).Msg("db init failed")
	}

	logger.GetLogger().Info().
		Str("filename", header.Filename).
		Str("hash", md5sum).
		Int64("original_size", originalSize).
		Int64("compressed_size", compressedSize).
		Str("compression", compressionType).
		Str("md5", md5sum).
		Str("mime", mimeType).
		Msg("file uploaded")

	resp := gin.H{
		"filename":          header.Filename,
		"original_size":     originalSize,
		"compressed_size":   compressedSize,
		"compression_type":  compressionType,
		"compression_ratio": float64(compressedSize) / float64(originalSize),
		"md5":               md5sum,
		"mime":              mimeType,
	}
	c.JSON(http.StatusOK, resp)
}

func downloadHandler(c *gin.Context) {
	filename := c.Param("filename")
	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}

	// Lookup metadata to find MD5
	db, err := ensureDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "database init failed"})
		return
	}
	var rec FileRecord
	if err := db.Where("filename = ?", filename).First(&rec).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	data, err := fsys.ReadObjectHashed(rec.MD5)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file content not found"})
		return
	}

	originalSize := int64(len(data))
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.FormatInt(originalSize, 10))
	c.Header("Content-Type", rec.MIME)
	c.Data(http.StatusOK, rec.MIME, data)

	logger.GetLogger().Info().Str("filename", filename).Str("hash", rec.MD5).Int64("size", originalSize).Msg("file downloaded")
}

func listHandler(c *gin.Context) {
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

	logger.GetLogger().Info().Int("count", len(files)).Msg("files listed")
	c.JSON(http.StatusOK, gin.H{"files": files, "count": len(files)})
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

	// Calculate logical statistics (based on metadata)
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
			uniqueCompressedSize += file.CompressedSize // first occurrence approximates physical (will align with on-disk hashed size)
		}
	}

	var compressionRatio float64
	if totalOriginalSize > 0 {
		compressionRatio = float64(totalCompressedSize) / float64(totalOriginalSize)
	}
	spaceSaved := totalOriginalSize - totalCompressedSize
	var spaceSavedPct float64
	if totalOriginalSize > 0 {
		spaceSavedPct = (float64(spaceSaved) / float64(totalOriginalSize)) * 100
	}

	// Physical storage scan (actual blobs on disk) to validate dedup numbers
	physicalObjectsCount := 0
	var physicalObjectsSize int64
	if fsys, err := fs.New(); err == nil {
		root := fsys.GetObjectsPath()
		_ = filepath.WalkDir(root, func(path string, d iofs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
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
	} // else ignore filesystem error; retain zero values

	// Dedup savings relative to logical compressed accumulation
	// logical compressed double-counts shared blobs, physicalObjectsSize is real disk usage
	var dedupSavedCompressed int64 = totalCompressedSize - physicalObjectsSize
	if dedupSavedCompressed < 0 {
		dedupSavedCompressed = 0
	}
	var dedupSavedCompressedPct float64
	if totalCompressedSize > 0 {
		dedupSavedCompressedPct = float64(dedupSavedCompressed) / float64(totalCompressedSize) * 100
	}
	// Savings versus original logical size
	var dedupSavedOriginal int64 = totalOriginalSize - physicalObjectsSize
	if dedupSavedOriginal < 0 {
		dedupSavedOriginal = 0
	}
	var dedupSavedOriginalPct float64
	if totalOriginalSize > 0 {
		dedupSavedOriginalPct = float64(dedupSavedOriginal) / float64(totalOriginalSize) * 100
	}

	logger.GetLogger().Info().
		Int("file_count", len(files)).
		Int("unique_hash_count", len(uniqueHashSeen)).
		Int64("logical_original", totalOriginalSize).
		Int64("logical_compressed", totalCompressedSize).
		Int64("physical_compressed", physicalObjectsSize).
		Float64("compression_ratio", compressionRatio).
		Msg("compression & dedup stats requested")

	c.JSON(http.StatusOK, gin.H{
		"file_count":               len(files),
		"unique_hash_count":        len(uniqueHashSeen),
		"total_original_size":      totalOriginalSize,
		"total_compressed_size":    totalCompressedSize,
		"compression_ratio":        compressionRatio,
		"space_saved":              spaceSaved,
		"space_saved_percentage":   spaceSavedPct,
		"compression_types":        compressionStats,
		"mime_types":               mimeStats,
		"unique_compressed_size":   uniqueCompressedSize,
		"physical_objects_count":   physicalObjectsCount,
		"physical_objects_size":    physicalObjectsSize,
		"dedup_saved_compressed":   dedupSavedCompressed,
		"dedup_saved_compr_pct":    dedupSavedCompressedPct,
		"dedup_saved_original":     dedupSavedOriginal,
		"dedup_saved_original_pct": dedupSavedOriginalPct,
	})
}

// uploadMultiHandler handles parallel multi-file uploads.
func uploadMultiHandler(c *gin.Context) {
	// Parse multipart form (use a large but bounded memory; rest to temp files)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB memory
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form"})
		return
	}
	form := c.Request.MultipartForm
	files := form.File["files"]
	if len(files) == 0 { // allow also field name "file" repeated
		files = form.File["file"]
	}
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})
		return
	}

	fsys, err := fs.New()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem init failed"})
		return
	}
	db, dbErr := ensureDB() // optional, continue without DB if fails

	type result struct {
		Filename        string  `json:"filename"`
		OriginalSize    int64   `json:"original_size"`
		CompressedSize  int64   `json:"compressed_size"`
		CompressionType string  `json:"compression_type"`
		CompressionRatio float64 `json:"compression_ratio"`
		MD5             string  `json:"md5"`
		MIME            string  `json:"mime"`
		Error           string  `json:"error,omitempty"`
	}

	results := make([]result, len(files))
	var wg sync.WaitGroup
	concurrency := 4
	sem := make(chan struct{}, concurrency)

	for i, fh := range files {
		wg.Add(1)
		index := i
		fileHeader := fh
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := &results[index]
			res.Filename = fileHeader.Filename

			f, err := fileHeader.Open()
			if err != nil {
				res.Error = "open failed"
				return
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				res.Error = "read failed"
				return
			}
			res.OriginalSize = int64(len(data))
			res.MD5 = file.MD5Sum(data)
			res.MIME = file.DetectMIME(data, fileHeader.Filename)
			preCT := compress.IsCompressedOrMIME(data, res.MIME)

			if err := fsys.WriteObjectHashedWithMIME(res.MD5, data, res.MIME); err != nil {
				res.Error = "store failed"
				return
			}
			cs, err := fsys.GetHashedObjectSize(res.MD5)
			if err != nil {
				cs = res.OriginalSize
			}
			res.CompressedSize = cs
			if preCT != compress.None {
				res.CompressionType = preCT.String()
			} else {
				res.CompressionType = fsys.GetCompressor().Type().String()
			}
			if res.OriginalSize > 0 {
				res.CompressionRatio = float64(res.CompressedSize) / float64(res.OriginalSize)
			}

			if dbErr == nil && db != nil {
				rec := &FileRecord{
					Filename:        res.Filename,
					Size:            res.OriginalSize,
					CompressedSize:  res.CompressedSize,
					CompressionType: res.CompressionType,
					MD5:             res.MD5,
					MIME:            res.MIME,
				}
				_ = db.Create(rec).Error // ignore individual errors here
			}

			logger.GetLogger().Info().Str("filename", res.Filename).Str("hash", res.MD5).Int64("original_size", res.OriginalSize).Int64("compressed_size", res.CompressedSize).Str("compression", res.CompressionType).Str("mime", res.MIME).Msg("file uploaded (multi)")
		}()
	}

	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"results": results, "count": len(results)})
}
