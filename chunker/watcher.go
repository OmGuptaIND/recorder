package chunker

import (
	"context"
	"log"

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
	fsWatcher, err := fsnotify.NewWatcher()

	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		ctx,
		make(chan struct{}, 1),
		fsWatcher,
	}

	log.Println("Watcher created")

	go watcher.startWatchEvents()

	return watcher, nil
}

func (w *Watcher) Stop() {
	log.Println("Stopping watcher")

	if w.fsWatcher != nil {
		w.fsWatcher.Close()
	}

	close(w.chunkChan)
}

// AddRecordingFolder adds a folder to the watcher.
func (w *Watcher) AddRecordingFolder(path string) error {
	return w.fsWatcher.Add(path)
}

// StartWatchEvents starts watching the events.
func (w *Watcher) startWatchEvents() {
	for {
		select {
		case event := <-w.fsWatcher.Events:
			log.Println("event: ", event)

			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("created file: ", event.Name, event)
			}

			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("modified file: ", event.Name)
			}

			if event.Op&fsnotify.Remove == fsnotify.Remove {
				log.Println("removed file: ", event.Name)
			}

			if event.Op&fsnotify.Chmod == fsnotify.Chmod {
				log.Println("changed permissions file: ", event.Name)
			}

		case err := <-w.fsWatcher.Errors:
			log.Println("watcher error: ", err)
		case <-w.ctx.Done():
			log.Println("watcher gorutine context done")
			return
		}
	}
}
