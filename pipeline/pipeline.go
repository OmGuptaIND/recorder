package pipeline

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/OmGuptaIND/config"
	"github.com/OmGuptaIND/display"
	"github.com/OmGuptaIND/livestream"
	"github.com/OmGuptaIND/recorder"
)

type NewPipelineOptions struct {
	PageUrl   string
	StreamUrl string
}

type Pipeline struct {
	ID         string
	Display    *display.Display
	Recorder   *recorder.Recorder
	Livestream *livestream.Livestream

	Ctx    context.Context
	cancel context.CancelFunc

	mtx *sync.Mutex
	Wg  *sync.WaitGroup

	*NewPipelineOptions
}

// NewPipeline initializes a new Pipeline with the specified options.
func NewPipeline(opts NewPipelineOptions) (*Pipeline, error) {
	ctx, cancel := context.WithCancel(context.Background())

	ID := fmt.Sprintf("pipeline_%d", time.Now().UTC().UnixMilli())

	pipeLine := &Pipeline{
		ID:                 ID,
		Ctx:                ctx,
		cancel:             cancel,
		Wg:                 &sync.WaitGroup{},
		mtx:                &sync.Mutex{},
		NewPipelineOptions: &opts,
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

	if _, err := display.LaunchChrome(p.PageUrl); err != nil {
		return fmt.Errorf("error Launching Chrome: %w", err)
	}

	p.Display = display

	return nil
}

// setupRecording: sets up the Recording.
func (p *Pipeline) setupRecording() error {
	recorder := recorder.NewRecorder(
		recorder.NewRecorderOptions{
			ID:             p.ID,
			Ctx:            p.Ctx,
			Wg:             p.Wg,
			Display:        p.Display,
			ShowFfmpegLogs: false,
		},
	)

	if err := recorder.StartRecording(); err != nil {
		return fmt.Errorf("error Starting Recording: %w", err)
	}

	p.Recorder = recorder

	return nil
}

// setupLivestream: sets up the Livestream.
func (p *Pipeline) setupLivestream() error {
	if p.StreamUrl == "" {
		return nil
	}

	l := livestream.NewLivestream(
		livestream.NewLivestreamOptions{
			Ctx:            p.Ctx,
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
func (p *Pipeline) Stop() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	log.Println("Stopping Pipeline...", p.ID)

	// Closes the context and all the resources, with it.
	p.cancel()

	if p.Display != nil {
		p.Display.Close()
	}

	p.Wg.Wait()
	log.Println("Pipeline Stopped", p.ID)

	return nil
}
