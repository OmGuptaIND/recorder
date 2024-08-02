package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerOpts struct {
	Level string
}

// New creates a new logger
func New(opts LoggerOpts) *zap.Logger {
	// Logger initialization
	config := zap.NewProductionConfig()

	level := "info"

	if opts.Level != "" {
		level = opts.Level
	}

	if l, err := zapcore.ParseLevel(level); err == nil {
		config.Level = zap.NewAtomicLevelAt(l)
	} else {
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := config.Build()

	if err != nil {
		panic(err)
	}

	return logger
}
