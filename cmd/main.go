package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/OmGuptaIND/api"
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

	appCtx := createAppContext(ctx, appStore)

	apiServer := api.NewApiServer(appCtx, api.ApiServerOptions{
		Port: 3000,
	})

	apiServer.Start()

	// Handle signals
	sig := pkg.HandleSignal()

	go func() {
		for val := range sig {
			if val == syscall.SIGINT || val == syscall.SIGTERM {
				log.Println("Shutting down...")
				cancel()
				signal.Stop(sig)
				return
			}
		}
	}()

	<-apiServer.Done()
}

// CreateGlobalContext creates a new context with the provided store and chunker
func createAppContext(ctx context.Context, store *store.AppStore) context.Context {
	ctx = context.WithValue(ctx, config.StoreKey, store)
	return ctx
}
