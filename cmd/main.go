package main

import (
	"context"
	"log"
	"sync"
	"syscall"

	"github.com/OmGuptaIND/api"
	"github.com/OmGuptaIND/config"
	"github.com/OmGuptaIND/display"
	"github.com/OmGuptaIND/pkg"
	store "github.com/OmGuptaIND/store"
)

var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a new store
	store.NewStore()

	// Create a new display
	display := display.NewDisplay(config.DEFAULT_DISPLAY_OPTS)

	if err := display.LaunchXvfb(); err != nil {
		log.Panicln(err)
	}

	defer display.Close()

	// Create a new API server
	apiServer := api.NewApiServer(api.ApiServerOptions{
		Port:    3000,
		Ctx:     ctx,
		Display: display,
	})

	// Start the API server
	wg.Add(1)
	apiServer.Start()
	defer func() {
		apiServer.Close()
		wg.Done()
	}()

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
