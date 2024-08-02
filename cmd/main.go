package main

import (
	"github.com/OmGuptaIND/logger"
)

func main() {
	logger := logger.New(logger.LoggerOpts{
		Level: "trace",
	})

	logger.Info("Hello, World!")
}
