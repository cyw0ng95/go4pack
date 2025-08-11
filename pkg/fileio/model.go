package fileio

import (
	"time"

	"gorm.io/gorm"

	"go4pack/pkg/common/database"
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
	AnalysisStatus  string         `json:"analysis_status" gorm:"default:pending"`
	AnalysisError   *string        `json:"analysis_error,omitempty"`
}

// ElfAnalyzeCached stores cached ELF analysis JSON for a file
type ElfAnalyzeCached struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FileID    uint      `gorm:"uniqueIndex" json:"file_id"`
	Data      string    `gorm:"type:text" json:"data"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GzipAnalyzeCached stores cached gzip (and optional tar) analysis JSON
type GzipAnalyzeCached struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FileID    uint      `gorm:"uniqueIndex" json:"file_id"`
	Data      string    `gorm:"type:text" json:"data"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RpmAnalyzeCached stores cached RPM header analysis JSON
type RpmAnalyzeCached struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FileID    uint      `gorm:"uniqueIndex" json:"file_id"`
	Data      string    `gorm:"type:text" json:"data"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ensureDB migrates and returns db (always AutoMigrate to add new columns)
func ensureDB() (*gorm.DB, error) {
	if db := database.Get(); db != nil {
		_ = db.AutoMigrate(&FileRecord{}, &ElfAnalyzeCached{}, &GzipAnalyzeCached{}, &RpmAnalyzeCached{})
		return db, nil
	}
	db, err := database.Init("filemeta.db", &FileRecord{}, &ElfAnalyzeCached{}, &GzipAnalyzeCached{}, &RpmAnalyzeCached{})
	if err != nil {
		return nil, err
	}
	_ = db.AutoMigrate(&FileRecord{}, &ElfAnalyzeCached{}, &GzipAnalyzeCached{}, &RpmAnalyzeCached{})
	return db, nil
}
