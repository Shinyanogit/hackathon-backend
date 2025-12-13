package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/shinyyama/hackathon-backend/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func BuildDSN(cfg *config.Config) string {
	addr := cfg.DBHost

	// Prefer Cloud SQL unix socket when INSTANCE_CONNECTION_NAME is provided.
	if cfg.InstanceConnectionName != "" {
		addr = fmt.Sprintf("unix(/cloudsql/%s)", cfg.InstanceConnectionName)
	} else if strings.HasPrefix(cfg.DBHost, "tcp(") {
		// already includes tcp()
	} else if strings.HasPrefix(cfg.DBHost, "unix(") {
		// already includes unix()
	} else if strings.HasPrefix(cfg.DBHost, "/") {
		addr = fmt.Sprintf("unix(%s)", cfg.DBHost)
	} else {
		addr = fmt.Sprintf("tcp(%s:%s)", cfg.DBHost, cfg.DBPort)
	}

	return fmt.Sprintf("%s:%s@%s/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.DBUser, cfg.DBPassword, addr, cfg.DBName)
}

func Connect(cfg *config.Config) (*gorm.DB, error) {
	dsn := BuildDSN(cfg)
	gcfg := &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.Default.LogMode(logger.Warn),
	}
	db, err := gorm.Open(mysql.Open(dsn), gcfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(10)

	return db, nil
}
