package api

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/gofiber/fiber/v3"
)

// ApiServerOptions defines the configuration options for the ApiServer.
type ApiServerOptions struct {
	Port int
	Wg   *sync.WaitGroup
	Ctx  context.Context
}

// ApiServer represents an HTTP server that provides endpoints to manage media nodes.
type ApiServer struct {
	app  *fiber.App
	opts ApiServerOptions
}

// ResultStruct defines the JSON structure for node results.
type ResultStruct struct {
	Status string `json:"status"`
	MomoId string `json:"momoId"`
	Region string `badgerholdIndex:"IdxRegion" json:"region"`
}

// NewApiServer initializes a new API server with the specified options.
func NewApiServer(opts ApiServerOptions) *ApiServer {
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	apiServer := &ApiServer{
		app:  app,
		opts: opts,
	}

	app.Get("/ping", apiServer.pingHandler)
	app.Use(apiServer.notFoundHandler)

	return apiServer
}

// `helloHandler` responds with a simple greeting.
func (a *ApiServer) pingHandler(c fiber.Ctx) error {
	return c.SendString("pong")
}

// errorHandler handles all internal server errors.
func errorHandler(c fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "Internal Server Error"
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		msg = e.Message
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
	log.Printf("Error %d: %s\n", code, msg)
	return c.Status(code).SendString(msg)
}

// `notFoundHandler` handles unmatched routes.
func (a *ApiServer) notFoundHandler(c fiber.Ctx) error {
	return fiber.NewError(fiber.StatusNotFound, "Resource not found")
}

// Start begins listening on the configured port.
func (a *ApiServer) Start() <-chan struct{} {
	addr := fmt.Sprintf(":%d", a.opts.Port)
	startedChan := make(chan struct{})

	go func() {
		err := a.app.Listen(addr, fiber.ListenConfig{
			ListenerNetwork:       "tcp",
			DisableStartupMessage: true,
			GracefulContext:       a.opts.Ctx,
			OnShutdownError: func(err error) {
				log.Printf("Error shutting down the server: %v\n", err)
				if a.opts.Wg != nil {
					a.opts.Wg.Done()
				}
			},
			OnShutdownSuccess: func() {
				if a.opts.Wg != nil {
					a.opts.Wg.Done()
				}
			},
			ListenerAddrFunc: func(net.Addr) {
				log.Printf("apiServer listening on :%d \n", a.opts.Port)
				close(startedChan)
			},
		})

		if err != nil {
			log.Printf("Error starting the server: %v\n", err)
			close(startedChan)
		}
	}()

	return startedChan
}

// Close gracefully shuts down the server.
func (a *ApiServer) Close() error {
	log.Println("closing the API server")

	return a.app.Shutdown()
}
