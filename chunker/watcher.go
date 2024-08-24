package chunker

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/OmGuptaIND/config"
	"github.com/OmGuptaIND/pkg"
	"github.com/fsnotify/fsnotify"
)

type ChunkInfo struct {
}

type Watcher struct {
	ctx       context.Context
	chunkChan chan struct{}
	fsWatcher *fsnotify.Watcher
}

// Watcher is responsible for watching the recording folder and chunking the recording.
func NewWatcher(ctx context.Context) (*Watcher, error) {
	if err := pkg.CreateDirectory(getRecordinDirectoryPath()); err != nil {
		return nil, err
	}

	watcher := &Watcher{
		ctx,
		make(chan struct{}, 1),
		nil,
	}

	log.Println("Watcher created")

	return watcher, nil
}

// Start starts the watcher.
func (w *Watcher) Start() error {
	fsWatcher, err := fsnotify.NewWatcher()

	if err != nil {
		return err
	}

	fsWatcher.Add(getRecordinDirectoryPath())

	w.fsWatcher = fsWatcher

	log.Println("Watcher Started Watching: ", getRecordinDirectoryPath())

	return nil
}

func (w *Watcher) Stop() {
	log.Println("Stopping watcher")

	if w.fsWatcher != nil {
		w.fsWatcher.Close()
	}

	close(w.chunkChan)
}

// GetRecordinDirectoryPath returns the path of the recording directory.
func getRecordinDirectoryPath() string {
	cwd, err := os.Getwd()

	if err != nil {
		log.Fatalf("Failed to get current working directory, %v", err)
	}

	return filepath.Join(cwd, config.RECORDING_DIR)
}
