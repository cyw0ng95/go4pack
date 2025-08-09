package fileio

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"go4pack/pkg/common/database"
	"go4pack/pkg/common/fs"
	"go4pack/pkg/common/logger"
	"go4pack/pkg/common/file"
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

	if err := fsys.WriteObject(header.Filename, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save file failed"})
		return
	}

	compressedSize, err := fsys.GetObjectSize(header.Filename)
	if err != nil {
		logger.GetLogger().Warn().Err(err).Str("filename", header.Filename).Msg("failed to get compressed size")
		compressedSize = originalSize
	}
	compressionType := fsys.GetCompressor().Type().String()

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

	// Check if file exists
	exists, err := fsys.ObjectExists(filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "filesystem error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Read the object (this will automatically decompress it)
	data, err := fsys.ReadObject(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Get original size for the response header
	originalSize := int64(len(data))

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.FormatInt(originalSize, 10))
	c.Header("Content-Type", "application/octet-stream")

	c.Data(http.StatusOK, "application/octet-stream", data)

	logger.GetLogger().Info().
		Str("filename", filename).
		Int64("size", originalSize).
		Msg("file downloaded")
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

	// Calculate statistics
	var totalOriginalSize, totalCompressedSize int64
	var compressionStats = make(map[string]int)

	for _, file := range files {
		totalOriginalSize += file.Size
		totalCompressedSize += file.CompressedSize
		compressionStats[file.CompressionType]++
	}

	var compressionRatio float64
	if totalOriginalSize > 0 {
		compressionRatio = float64(totalCompressedSize) / float64(totalOriginalSize)
	}

	spaceSaved := totalOriginalSize - totalCompressedSize

	logger.GetLogger().Info().
		Int("file_count", len(files)).
		Int64("total_original_size", totalOriginalSize).
		Int64("total_compressed_size", totalCompressedSize).
		Float64("compression_ratio", compressionRatio).
		Msg("compression stats requested")

	c.JSON(http.StatusOK, gin.H{
		"file_count":             len(files),
		"total_original_size":    totalOriginalSize,
		"total_compressed_size":  totalCompressedSize,
		"compression_ratio":      compressionRatio,
		"space_saved":            spaceSaved,
		"space_saved_percentage": (float64(spaceSaved) / float64(totalOriginalSize)) * 100,
		"compression_types":      compressionStats,
	})
}
