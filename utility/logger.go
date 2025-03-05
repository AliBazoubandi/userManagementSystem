package utility

import "go.uber.org/zap"

type Logger struct {
	Logger *zap.Logger
}

var AppLogger Logger

func Init() {
	logger, _ := zap.NewDevelopment()
	AppLogger = Logger{Logger: logger}
	defer func(logger *zap.Logger) {
		err := logger.Sync()
		if err != nil {
			logger.Warn("Failed to sync logger", zap.Error(err))
		}
	}(logger)
}
