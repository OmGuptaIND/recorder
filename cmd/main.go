package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/OmGuptaIND/api"
	"github.com/OmGuptaIND/env"
	"github.com/OmGuptaIND/pkg"
	store "github.com/OmGuptaIND/store"
)

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

	apiServer.Start()

	// Handle signals
	sig := pkg.HandleSignal()

	go func() {
		for val := range sig {
			if val == syscall.SIGINT || val == syscall.SIGTERM {
				cancel()
				signal.Stop(sig)
				return
			}
		}
	}()

	<-apiServer.Done()
}
