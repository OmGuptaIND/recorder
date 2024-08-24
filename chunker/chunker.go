package chunker

import (
	"context"

	"github.com/OmGuptaIND/cloud"
)

type ChunkerOptions struct{}

type Chunker struct {
	ctx         context.Context
	CloudClient cloud.CloudClient
	*ChunkerOptions
}

// Chunker is responsible for chunking the recording, takes the decision when to batch upload to the cloud.
// and then we to locally stitch the chunks together and upload to the cloud.
func NewChunker(ctx context.Context, opts *ChunkerOptions) (*Chunker, error) {
	cloudClient, err := cloud.NewAwsClient(ctx, &cloud.AwsClientOptions{})

	if err != nil {
		return nil, err
	}

	chunker := &Chunker{
		context.WithoutCancel(ctx),
		cloudClient,
		opts,
	}

	return chunker, nil
}

// StartChunking starts the chunking process.
func (c *Chunker) StartChunking() error {
	return nil
}
