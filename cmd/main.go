package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/OmGuptaIND/api"
	"github.com/OmGuptaIND/cloud"
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

	cloudClient, err := cloud.NewAwsClient(ctx, &cloud.AwsClientOptions{})

	if err != nil {
		log.Fatalf("Failed to create cloud client: %v", err)
	}

	appCtx := createAppContext(ctx, appStore, cloudClient)

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
func createAppContext(ctx context.Context, store *store.AppStore, client cloud.CloudClient) context.Context {
	ctx = context.WithValue(ctx, config.StoreKey, store)
	ctx = context.WithValue(ctx, config.CloudClientKey, client)

	return ctx
}
