package config

import (
	"go.uber.org/zap"
)

var Logger *zap.Logger

func init() {
	Logger, _ = zap.NewProduction()
}

func GetLogger() *zap.Logger {
	return Logger
}
