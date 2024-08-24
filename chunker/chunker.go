package chunker

import (
	"context"
	"log"
	"time"

	"github.com/OmGuptaIND/cloud"
	"github.com/OmGuptaIND/executor"
)

type ChunkerOptions struct{}

type Chunker struct {
	ctx         context.Context
	cloudClient cloud.CloudClient
	executor    *executor.WorkerExecutor

	done chan struct{}
	*ChunkerOptions
}

// Chunker is responsible for chunking the recording, takes the decision when to batch upload to the cloud.
// and then we to locally stitch the chunks together and upload to the cloud.
func NewChunker(ctx context.Context, opts *ChunkerOptions) (*Chunker, error) {
	cloudClient, err := cloud.NewAwsClient(ctx, &cloud.AwsClientOptions{})

	if err != nil {
		return nil, err
	}

	workerExecutor := executor.NewWorkerExecutor(ctx, &executor.WorkerExecutorOptions{
		MaxRetries:   3,
		WorkerCount:  5,
		RetryBackoff: 10 * time.Second,
	})

	chunker := &Chunker{
		context.WithoutCancel(ctx),
		cloudClient,
		workerExecutor,
		make(chan struct{}, 1),
		opts,
	}

	return chunker, nil
}

// EnqueueChunk enqueues a chunk to be uploaded to the cloud.
func (c *Chunker) Wait() {
	log.Println("Waiting for any pending chunk to finish uploading")
	c.executor.Wait()
}

// Stop stops the chunker.
func (c *Chunker) Stop() {
	c.executor.Stop()
}

// Done returns a channel that is closed when the chunker is done.
func (c *Chunker) Done() <-chan struct{} {
	return c.done
}

// StartChunking starts the chunking process.
func (c *Chunker) StartChunking() error {
	return nil
}
