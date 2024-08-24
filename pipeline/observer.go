package pipeline

import "context"

type ObserverOptions struct{}

type Observer struct {
	ctx context.Context
	*ObserverOptions
}

// NewMonitor creates a new Monitor, Which Keeps track of the pipeline.
func NewObserver(ctx context.Context, opts *ObserverOptions) *Observer {
	return &Observer{
		context.WithoutCancel(ctx),
		opts,
	}
}
