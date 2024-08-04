package pkg

import (
	"os"
	"os/signal"
	"syscall"
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
