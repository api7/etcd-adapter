package utils

import (
	"go.uber.org/zap"
)

var log *zap.Logger

func GetLogger() *zap.Logger {
	if log == nil {
		log = initLogger()
	}
	return log
}

func initLogger() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}
