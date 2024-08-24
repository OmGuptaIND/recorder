package pipeline

import "context"

type MonitorOptions struct{}

type Monitor struct {
	ctx context.Context
	*MonitorOptions
}

// NewMonitor creates a new Monitor, Which Keeps track of the pipeline.
func NewMonitor(ctx context.Context, opts *MonitorOptions) *Monitor {
	return &Monitor{
		context.WithoutCancel(ctx),
		opts,
	}
}
