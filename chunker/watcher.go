package chunker

import (
	"context"
	"log"
)

type ChunkInfo struct {
}

type Watcher struct {
	ctx       context.Context
	chunkChan chan struct{}
}

// Watcher is responsible for watching the recording folder and chunking the recording.
func NewWatcher(ctx context.Context) (*Watcher, error) {
	watcher := &Watcher{
		ctx,
		make(chan struct{}, 1),
	}

	log.Println("Watcher created")

	go watcher.startWatchEvents()

	return watcher, nil
}

func (w *Watcher) Stop() {
	log.Println("Stopping watcher")

	close(w.chunkChan)
}

// AddRecordingFolder adds a folder to the watcher.
func (w *Watcher) AddRecordingFolder(path string) error {
	return nil
}

// StartWatchEvents starts watching the events.
func (w *Watcher) startWatchEvents() {

}
