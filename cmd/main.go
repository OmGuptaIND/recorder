package main

import (
	"context"
	"sync"
	"syscall"

	"github.com/OmGuptaIND/api"
	"github.com/OmGuptaIND/pkg"
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
