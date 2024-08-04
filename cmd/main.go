package main

import (
	"context"
	"log"
	"sync"
	"syscall"

	"github.com/OmGuptaIND/api"
	"github.com/OmGuptaIND/display"
	"github.com/OmGuptaIND/pkg"
	"github.com/OmGuptaIND/recorder"
)

var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	apiServer := api.NewApiServer(api.ApiServerOptions{
		Port: 3000,
		Wg:   &wg,
		Ctx:  ctx,
	})

	wg.Add(1)
	apiServer.Start()
	defer apiServer.Close()

	display := display.NewDisplay(display.DisplayOptions{
		Width:   1280,
		Height:  720,
		Depth:   24,
		Display: ":99",
	})

	if err := display.Launch("https://giphy.com"); err != nil {
		log.Panicln(err)
	}

	defer display.Close()

	display.TakeScreenshot()

	recorder := recorder.NewRecorder(recorder.NewRecorderOptions{
		Width:   1280,
		Height:  720,
		Depth:   24,
		Display: ":99",
	})

	if err := recorder.StartRecording(); err != nil {
		log.Panicln(err)
	}

	defer recorder.StopRecording()

	// Handle signals
	sig := pkg.HandleSignal()

	go func() {
		for val := range sig {
			if val == syscall.SIGINT || val == syscall.SIGTERM {
				cancel()
				close(sig)
				return
			}
		}
	}()

	wg.Wait()
}
