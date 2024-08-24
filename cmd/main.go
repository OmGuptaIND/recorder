package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/OmGuptaIND/api"
	"github.com/OmGuptaIND/chunker"
	"github.com/OmGuptaIND/config"
	"github.com/OmGuptaIND/env"
	"github.com/OmGuptaIND/pkg"
	store "github.com/OmGuptaIND/store"
)

func main() {
	env.LoadEnvironmentVariables()

	appStore := store.NewStore()

	globalCtx := context.WithValue(context.Background(), config.StoreKey, appStore)

	ctx, cancel := context.WithCancel(globalCtx)
	defer cancel()

	recChunker, err := chunker.NewChunker(ctx, &chunker.ChunkerOptions{})
	defer recChunker.Stop()

	if err != nil {
		log.Fatalf("Error creating chunker: %v", err)
	}

	// Create a new API server
	apiServer := api.NewApiServer(ctx, api.ApiServerOptions{
		Port: 3000,
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
	recChunker.Wait()
}
