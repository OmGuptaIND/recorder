package chunker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/OmGuptaIND/cloud"
	"github.com/OmGuptaIND/config"
	"github.com/OmGuptaIND/executor"
	"github.com/OmGuptaIND/recorder"
)

type ChunkerOptions struct{}

type Chunker struct {
	ctx         context.Context
	cloudClient cloud.CloudClient
	executor    *executor.WorkerExecutor

	mtx       *sync.Mutex
	recorders map[string]*recorder.Recorder

	jobChan chan executor.Job

	done chan struct{}
	*ChunkerOptions
}

// GetChunker retrieves the chunker from the context, if ctx is nil it returns the global chunker.
func GetChunker(ctx *context.Context) *Chunker {
	if ctx == nil {
		return nil
	}

	chunker, _ := (*ctx).Value(config.ChunkerKey).(*Chunker)

	return chunker
}

// Chunker is responsible for chunking the recording, takes the decision when to batch upload to the cloud.
// and then we to locally stitch the chunks together and upload to the cloud.
func NewChunker(ctx context.Context, opts *ChunkerOptions) (*Chunker, error) {
	cloudClient, err := cloud.NewAwsClient(ctx, &cloud.AwsClientOptions{})

	if err != nil {
		log.Println("Error creating cloud client")
		return nil, err
	}

	workerExecutor := executor.NewWorkerExecutor(ctx, &executor.WorkerExecutorOptions{
		MaxRetries:   3,
		WorkerCount:  5,
		RetryBackoff: 10 * time.Second,
	})

	workerExecutor.Start()

	chunker := &Chunker{
		ctx:            ctx,
		cloudClient:    cloudClient,
		executor:       workerExecutor,
		mtx:            &sync.Mutex{},
		recorders:      make(map[string]*recorder.Recorder),
		jobChan:        make(chan executor.Job),
		done:           make(chan struct{}, 1),
		ChunkerOptions: opts,
	}

	log.Println("Chunker created")

	go chunker.listenJobEvents()

	return chunker, nil
}

// EnqueueChunk enqueues a chunk to be uploaded to the cloud.
func (c *Chunker) Wait() {
	log.Println("Waiting for any pending chunk to finish uploading, [ Else Close Forcefully ]")
	c.executor.Wait()
}

// Done returns a channel that is closed when the chunker is done.
func (c *Chunker) Done() <-chan struct{} {
	return c.done
}

// listenJobEvents listens for new chunk jobs.
func (c *Chunker) listenJobEvents() {
	for {
		select {
		case <-c.ctx.Done():
			log.Println("Chunker context done")
			return
		case job := <-c.jobChan:
			c.executor.Enqueue(job)
		}
	}
}

// EnqueueChunk enqueues a chunk to be uploaded to the cloud.
func (c *Chunker) AddRecorder(recorder *recorder.Recorder) {
	log.Println("Adding recorder to chunker")
	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.recorders[recorder.ID] = recorder

	log.Printf("Recorder %s added to chunker\n", recorder.ID)

	c.buildChunkPipeline(recorder)
}

// RemoveRecorder removes the recorder from the chunker.
func (c *Chunker) RemoveRecorder(recorder *recorder.Recorder) {
	log.Println("Removing recorder from chunker")
	c.mtx.Lock()
	defer c.mtx.Unlock()

	delete(c.recorders, recorder.ID)

	log.Println("Recorder removed from chunker", recorder.ID)
}

// BuildChunkPipeline builds the chunk pipeline for the recorder.
func (c *Chunker) buildChunkPipeline(recorder *recorder.Recorder) {
	log.Println("Building chunk pipeline for recorder", recorder.ID)

	go func() {
		for {
			select {
			case <-recorder.Watcher.Done():
				log.Println("Stopped Chunking Recorder Done, Recorder Watcher")
				go c.RemoveRecorder(recorder)
				return
			case chunk := <-recorder.Watcher.ChunkChan():
				log.Println("Received New chunk", chunk)

				jobCtx, cancel := context.WithTimeout(recorder.GetContext(), 20*time.Second)

				c.jobChan <- executor.Job{
					Ctx: jobCtx,
					Id:  chunk.RecorderID,
					JobFunc: func() error {
						defer cancel()
						return c.uploadChunk(jobCtx, chunk)
					},
					OnSuccess: func() {
						cancel()
						c.uploadSuccess(recorder.GetContext(), chunk)
					},
					OnError: func(err error) {
						cancel()
						c.uploadError(recorder.GetContext(), err, chunk)
					},
				}
			}
		}
	}()
}

// ChunkJobFunc is the function that is executed for each chunk job.
func (c *Chunker) uploadChunk(ctx context.Context, chunkInfo config.ChunkInfo) error {
	log.Println("Chunking job started", chunkInfo)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	filePath := fmt.Sprintf("%s/%s", chunkInfo.RecorderID, chunkInfo.ChunkName)

	// Upload the chunk to the cloud
	if err := c.cloudClient.UploadFile(&filePath, chunkInfo.ChunkPath); err != nil {
		log.Println("Error uploading chunk", err)
		return err
	}

	log.Println("Chunking job completed", chunkInfo)

	return nil
}

func (c *Chunker) uploadSuccess(ctx context.Context, chunkInfo config.ChunkInfo) {
	log.Println("Chunking job success", chunkInfo)

	if ctx.Err() != nil {
		return
	}

	recorder := c.recorders[chunkInfo.RecorderID]

	if recorder == nil {
		log.Println("Recorder not found for chunk", chunkInfo)
		return
	}

	recorder.Watcher.ChunkUploadSucess(chunkInfo)
}

func (c *Chunker) uploadError(ctx context.Context, err error, chunkInfo config.ChunkInfo) {
	log.Println("Chunking job error", chunkInfo)

	if ctx.Err() != nil {
		return
	}

	recorder := c.recorders[chunkInfo.RecorderID]

	if recorder == nil {
		log.Println("Recorder not found for chunk", chunkInfo)
		return
	}

	recorder.Watcher.ChunkUploadFailed(err, chunkInfo)
}

// Stop stops the chunker.
func (c *Chunker) Stop() {
	log.Println("Stopping chunker")

}
