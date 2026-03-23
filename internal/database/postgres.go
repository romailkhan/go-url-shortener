package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a Postgres connection pool tuned for low latency.
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		Logger:                 logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sql db: %w", err)
	}
	sqlDB.SetMaxIdleConns(8)
	sqlDB.SetMaxOpenConns(32)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// Migrate runs AutoMigrate for the given models.
func Migrate(db *gorm.DB, models ...any) error {
	return db.AutoMigrate(models...)
}
