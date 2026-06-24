package logs

import (
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	config := zap.NewDevelopmentConfig()
	// config.EncoderConfig.TimeKey = "timestamp"
	// config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	var err error
	logger, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
}

func Sync() error {
	return logger.Sync()
}

func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

func Warn(msg any, fields ...zap.Field) {
	switch v := msg.(type) {
	case error:
		logger.Warn(v.Error(), fields...)
	case string:
		logger.Warn(v, fields...)
	}
}

func Error(msg any, fields ...zap.Field) {
	switch v := msg.(type) {
	case error:
		logger.Error(v.Error(), fields...)
	case string:
		logger.Error(v, fields...)
	}
}
