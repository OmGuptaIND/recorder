package recorder

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/OmGuptaIND/config"
)

type WatcherOptions struct {
	recorder *Recorder
}

type Watcher struct {
	ctx          context.Context
	chunkCounter int
	chunkChan    chan config.ChunkInfo
	done         chan bool
	*WatcherOptions
}

// ChunkChan returns the channel for the chunks.
func (w *Watcher) ChunkChan() <-chan config.ChunkInfo {
	return w.chunkChan
}

// Watcher is responsible for watching the recording folder and chunking the recording.
func NewWatcher(ctx context.Context, opts *WatcherOptions) *Watcher {
	watcher := &Watcher{
		ctx,
		0,
		make(chan config.ChunkInfo, 20),
		make(chan bool, 1),
		opts,
	}

	log.Println("Watcher created")

	return watcher
}

// Done returns the done channel for the Watcher.
func (w *Watcher) Done() <-chan bool {
	return w.done
}

// Start the Watcher functionality
func (w *Watcher) Start() error {
	if w.ctx.Err() != nil {
		return w.ctx.Err()
	}

	go func() {
		for {
			timer := time.NewTicker(12 * time.Second)
			select {
			case <-w.ctx.Done():
				log.Println("Context is done, Returning from Watcher")
				timer.Stop()
				return
			case <-timer.C:
				w.grabChunks()
			}
		}
	}()

	return nil
}

// grabChunks grabs the chunks from the recording directory and sends them to the chunk channel.
func (w *Watcher) grabChunks() {
	for i := w.chunkCounter; ; i++ {
		chunkName := fmt.Sprintf("%s%05d%s", "chunk_", w.chunkCounter, ".mp4")

		if w.ctx.Err() != nil {
			log.Println("Context is done, Returning from Watcher")
			return
		}

		chunkPath := filepath.Join(w.recorder.GetRecordinDirectoryPath(), chunkName)

		chunkFile, err := os.Stat(chunkPath)

		if err == nil || !os.IsNotExist(err) {
			log.Println("Chunk Found: ", chunkPath)
			chunkInfo := config.ChunkInfo{
				RecorderID: w.recorder.ID,
				ChunkName:  chunkName,
				ChunkPath:  chunkPath,
				ChunkSize:  chunkFile.Size(),
			}

			w.chunkChan <- chunkInfo
			w.chunkCounter++
			continue
		}
		break
	}
}

// Stop the Watcher functionality
func (w *Watcher) Stop() {
	log.Println("Stopping watcher")

	if w.ctx.Err() != nil {
		return
	}

	close(w.chunkChan)
}

func (w *Watcher) ChunkUploadSucess(chunkInfo config.ChunkInfo) {
	log.Println("Chunk Upload Success")

	if w.ctx.Err() != nil {
		log.Println("Context is done, Returning from ChunkUploadSucess")
		return
	}

	w.chunkChan <- chunkInfo
}

func (w *Watcher) ChunkUploadFailed(err error, chunkInfo config.ChunkInfo) {
	log.Println("Chunk Upload Failed", err)

	if w.ctx.Err() != nil {
		log.Println("Context is done, Returning from ChunkUploadFailed")
		return
	}

	w.chunkChan <- chunkInfo
}

// Close closes the Wathcer for the recording, uplods the pending chunks and stops the watcher.
func (w *Watcher) Close() {
	close(w.chunkChan)
}
