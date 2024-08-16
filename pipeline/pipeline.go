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

	context context.Context
	cancel  context.CancelFunc

	mtx *sync.Mutex
	Wg  *sync.WaitGroup

	*NewPipelineOptions
}

// NewPipeline initializes a new Pipeline with the specified options.
func NewPipeline(opts NewPipelineOptions) (*Pipeline, error) {
	context, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	ID := fmt.Sprintf("pipeline_%d", time.Now().UTC().UnixMilli())

	display := display.NewDisplay(display.DisplayOptions{
		ID:     ID,
		Wg:     wg,
		Width:  config.DEFAULT_DISPLAY_OPTS.Width,
		Height: config.DEFAULT_DISPLAY_OPTS.Height,
		Depth:  config.DEFAULT_DISPLAY_OPTS.Depth,
	})

	err := display.LaunchXvfb()

	defer func() {
		if err != nil {
			log.Println("Error Occured Starting Pipeline, Closing Display", err)
			cancel()

			if display != nil {
				display.Close()
			}
		}
	}()

	if err != nil {
		log.Println("Error Occured Launching XVFB, Closing Display", err)
		return nil, err
	}

	err = display.LaunchPulseSink()

	if err != nil {
		log.Println("Error Occured Launching Pulse Sink, Closing Display", err)
		return nil, err
	}

	_, err = display.LaunchChrome(opts.PageUrl)

	if err != nil {
		log.Println("Error Occured Launching Chrome, Closing Display", err)
		return nil, err
	}

	time.Sleep(time.Second * 3)

	recorder := recorder.NewRecorder(
		recorder.NewRecorderOptions{
			ID:             ID,
			Ctx:            context,
			Wg:             wg,
			Display:        display,
			ShowFfmpegLogs: false,
		},
	)

	if err := recorder.StartRecording(); err != nil {
		return nil, err
	}

	pipeLine := &Pipeline{
		ID:                 ID,
		context:            context,
		cancel:             cancel,
		Wg:                 wg,
		mtx:                &sync.Mutex{},
		NewPipelineOptions: &opts,
	}

	if opts.StreamUrl != "" {
		l := livestream.NewLivestream(
			livestream.NewLivestreamOptions{
				Ctx:            context,
				Wg:             wg,
				ShowFfmpegLogs: false,
				StreamUrl:      opts.StreamUrl,
				Display:        display,
			},
		)

		if err := l.StartStream(); err != nil {
			return nil, err
		}

		pipeLine.Livestream = l
	}

	return pipeLine, nil
}

// Stop: stops the Pipeline.
func (p *Pipeline) Stop() error {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	log.Println("Stopping Pipeline...", p.ID)

	p.cancel()

	if p.Display != nil {
		p.Display.Close()
	}

	p.Wg.Wait()

	log.Println("Pipeline Stopped", p.ID)

	return nil
}
