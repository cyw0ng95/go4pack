package database

import (
	"fmt"
	"path/filepath"
	"sync"

	"go4pack/pkg/common/fs"
	"go4pack/pkg/common/logger"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	instance *gorm.DB
	once     sync.Once
)

// Init initializes the sqlite database inside .runtime directory
func Init(dbName string, models ...interface{}) (*gorm.DB, error) {
	var initErr error
	once.Do(func() {
		fsys, err := fs.New()
		if err != nil {
			initErr = fmt.Errorf("filesystem init failed: %w", err)
			return
		}
		dbPath := filepath.Join(fsys.GetRuntimePath(), dbName)
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			initErr = fmt.Errorf("open db failed: %w", err)
			return
		}
		// Auto migrate models
		if len(models) > 0 {
			if err := db.AutoMigrate(models...); err != nil {
				initErr = fmt.Errorf("auto migrate failed: %w", err)
				return
			}
		}
		instance = db
		logger.GetLogger().Info().Str("db", dbPath).Msg("database initialized")
	})
	return instance, initErr
}

// Get returns the gorm DB instance
func Get() *gorm.DB { return instance }
