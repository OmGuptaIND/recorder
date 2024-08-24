package pkg

import (
	"fmt"
	"log"
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

// CreateDirectory creates a directory if it does not exist.
func CreateDirectory(directoryPath string) error {

	if _, err := os.Stat(directoryPath); os.IsNotExist(err) {
		if err := os.MkdirAll(directoryPath, 0755); err != nil {
			return err
		}

		log.Printf("Created directory at %s", directoryPath)
	}

	return nil
}
