package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// New creates a new GORM database connection using the provided DSN.
func New(dsn string) (*gorm.DB, error) {
	gormLogger := logger.Default.LogMode(logger.Info)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)

	log.Println("connected to database")
	return db, nil
}
