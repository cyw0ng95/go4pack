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
	ElfAnalysis     *string        `json:"elf_analysis,omitempty"` // ELF analysis JSON (if applicable)
}

// ensureDB migrates and returns db (always AutoMigrate to add new columns)
func ensureDB() (*gorm.DB, error) {
	if db := database.Get(); db != nil {
		_ = db.AutoMigrate(&FileRecord{})
		return db, nil
	}
	db, err := database.Init("filemeta.db", &FileRecord{})
	if err != nil {
		return nil, err
	}
	_ = db.AutoMigrate(&FileRecord{})
	return db, nil
}
