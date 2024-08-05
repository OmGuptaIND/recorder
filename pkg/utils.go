package pkg

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// handleSignal listens for signals and closes the badger node when a signal is received.
func HandleSignal() chan os.Signal {
	signalChan := make(chan os.Signal, 20)
	signal.Notify(
		signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGHUP,
	)

	return signalChan
}

func RandomDisplay() string {
	return fmt.Sprintf(":%d", (time.Now().Nanosecond()%1000)+os.Getpid()%1000+100)
}
