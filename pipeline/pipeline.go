package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/OmGuptaIND/cloud"
	"github.com/OmGuptaIND/config"
	"github.com/OmGuptaIND/display"
	"github.com/OmGuptaIND/livestream"
	"github.com/OmGuptaIND/recorder"
	"github.com/OmGuptaIND/uploader"
)

type NewPipelineOptions struct {
	RecordUrl string
	StreamUrl string
}

type Pipeline struct {
	ctx    context.Context
	cancel context.CancelFunc

	ID         string
	Display    *display.Display
	Recorder   *recorder.Recorder
	Uploader   *uploader.Uploader
	Livestream *livestream.Livestream

	mtx *sync.Mutex
	Wg  *sync.WaitGroup

	*NewPipelineOptions
}

// NewPipeline initializes a new Pipeline with the specified options.
func NewPipeline(ctx context.Context, opts *NewPipelineOptions) (*Pipeline, error) {
	ctx, cancel := context.WithCancel(ctx)

	ID := fmt.Sprintf("pipeline_%d", time.Now().UTC().UnixMilli())

	pipeLine := &Pipeline{
		ID:                 ID,
		ctx:                ctx,
		cancel:             cancel,
		Wg:                 &sync.WaitGroup{},
		mtx:                &sync.Mutex{},
		NewPipelineOptions: opts,
	}

	return pipeLine, nil
}

// Start: starts the Pipeline.
func (p *Pipeline) Start() error {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Recovered from panic", err)
			p.Stop()
		}
	}()

	if err := p.setupDisplay(); err != nil {
		return err
	}

	if err := p.setupRecording(); err != nil {
		return err
	}

	if err := p.setupLivestream(); err != nil {
		return err
	}

	return nil
}

// setupDisplay: sets up the Display.
func (p *Pipeline) setupDisplay() error {
	display := display.NewDisplay(display.DisplayOptions{
		ID:     p.ID,
		Wg:     p.Wg,
		Width:  config.DEFAULT_DISPLAY_OPTS.Width,
		Height: config.DEFAULT_DISPLAY_OPTS.Height,
		Depth:  config.DEFAULT_DISPLAY_OPTS.Depth,
	})

	if err := display.LaunchXvfb(); err != nil {
		return fmt.Errorf("error Launching XVFB: %w", err)
	}

	if err := display.LaunchPulseSink(); err != nil {
		return fmt.Errorf("error Launching Pulse Sink: %w", err)
	}

	if _, err := display.LaunchChrome(p.RecordUrl); err != nil {
		return fmt.Errorf("error Launching Chrome: %w", err)
	}

	p.Display = display

	return nil
}

// setupRecording: sets up the Recording.
func (p *Pipeline) setupRecording() error {
	recorder, err := recorder.NewRecorder(
		p.ctx,
		recorder.NewRecorderOptions{
			ID:             p.ID,
			Wg:             p.Wg,
			Display:        p.Display,
			ShowFfmpegLogs: false,
		},
	)

	if err != nil {
		return fmt.Errorf("error Creating Recorder: %w", err)
	}

	if err := recorder.StartRecording(); err != nil {
		return fmt.Errorf("error Starting Recording: %w", err)
	}

	p.Recorder = recorder

	uploader, err := uploader.NewUploader(
		p.ctx,
		recorder.GetReader(),
		&recorder.ID,
	)

	if err != nil {
		return fmt.Errorf("error Creating Uploader: %w", err)
	}

	p.Uploader = uploader

	p.Wg.Add(1)

	go func() {
		defer p.Wg.Done()

		if err := p.Uploader.Start(); err != nil {
			log.Println("Error Starting Uploader", err)
		}
	}()

	return nil
}

// setupLivestream: sets up the Livestream.
func (p *Pipeline) setupLivestream() error {
	if p.StreamUrl == "" {
		return nil
	}

	l := livestream.NewLivestream(
		p.ctx,
		livestream.NewLivestreamOptions{
			Wg:             p.Wg,
			ShowFfmpegLogs: false,
			StreamUrl:      p.StreamUrl,
			Display:        p.Display,
		},
	)

	if err := l.StartStream(); err != nil {
		return fmt.Errorf("error Starting Livestream: %w", err)
	}

	p.Livestream = l

	return nil
}

// Stop: stops the Pipeline.
func (p *Pipeline) Stop() (*cloud.CloudUploadPartCompleted, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	log.Println("Stopping Pipeline...", p.ID)

	p.cancel()

	if p.Display != nil {
		p.Display.Close()
	}

	resp, err := p.Uploader.Stop()

	if err != nil {
		return nil, fmt.Errorf("error Stopping Uploader: %w", err)
	}

	p.Wg.Wait()
	log.Println("Pipeline Stopped", p.ID)

	return resp, nil
}
