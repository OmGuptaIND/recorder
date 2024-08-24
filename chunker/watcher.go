package chunker

import (
	"context"
	"log"
	"sync"
)

type ChunkInfo struct {
}

type Watcher struct {
	ctx       context.Context
	wg        *sync.WaitGroup
	chunkChan chan struct{}
}

// Watcher is responsible for watching the recording folder and chunking the recording.
func NewWatcher(ctx context.Context) (*Watcher, error) {
	watcher := &Watcher{
		ctx,
		&sync.WaitGroup{},
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
