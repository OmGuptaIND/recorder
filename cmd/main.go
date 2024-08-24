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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	appStore := store.NewStore()

	recChunker, err := chunker.NewChunker(context.WithValue(ctx, config.StoreKey, appStore), &chunker.ChunkerOptions{})
	defer recChunker.Stop()

	if err != nil {
		log.Fatalf("Error creating chunker: %v", err)
	}

	appCtx := createAppContext(ctx, appStore, recChunker)

	apiServer := api.NewApiServer(appCtx, api.ApiServerOptions{
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

// CreateGlobalContext creates a new context with the provided store and chunker
func createAppContext(ctx context.Context, store *store.AppStore, chunker *chunker.Chunker) context.Context {
	ctx = context.WithValue(ctx, config.StoreKey, store)
	ctx = context.WithValue(ctx, config.ChunkerKey, chunker)
	return ctx
}
