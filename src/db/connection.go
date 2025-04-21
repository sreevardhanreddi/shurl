package db

import (
	"database/sql"
	"os"
	"shurl/src/config"
	"time"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func Connect() (*sql.DB, error) {
	logger := config.GetLogger()

	db, err := sql.Open("postgres", os.Getenv("POSTGRES_URI"))
	if err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		return nil, err
	} else {
		logger.Info("Connected to database")
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping database", zap.Error(err))
		return nil, err
	}

	logger.Info("Connected to database")

	return db, nil
}
