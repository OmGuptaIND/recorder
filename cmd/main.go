package main

import (
	"context"
	"sync"
	"syscall"

	"github.com/OmGuptaIND/api"
	"github.com/OmGuptaIND/env"
	"github.com/OmGuptaIND/pkg"
	store "github.com/OmGuptaIND/store"
)

var wg sync.WaitGroup

func main() {
	env.LoadEnvironmentVariables()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a new store
	store.NewStore()

	// Create a new API server
	apiServer := api.NewApiServer(api.ApiServerOptions{
		Port: 3000,
		Ctx:  ctx,
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
