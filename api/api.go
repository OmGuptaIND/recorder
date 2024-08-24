package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/OmGuptaIND/pipeline"
	"github.com/OmGuptaIND/recorder"
	"github.com/OmGuptaIND/store"
	"github.com/gofiber/fiber/v3"
)

// ApiServerOptions defines the configuration options for the ApiServer.
type ApiServerOptions struct {
	Port int
	Wg   *sync.WaitGroup
}

// ApiServer represents an HTTP server that provides endpoints to manage media nodes.
type ApiServer struct {
	ctx  context.Context
	app  *fiber.App
	opts ApiServerOptions
	done chan bool
}

// NewApiServer initializes a new API server with the specified options.
func NewApiServer(ctx context.Context, opts ApiServerOptions) *ApiServer {
	app := fiber.New(fiber.Config{
		ErrorHandler: errorHandler,
	})

	apiServer := &ApiServer{
		app:  app,
		opts: opts,
		done: make(chan bool, 1),
	}

	app.Get("/ping", apiServer.pingHandler)
	app.Post("/start-recording", apiServer.startRecording)
	app.Patch("/stop-recording", apiServer.stopRecording)
	app.Use(apiServer.notFoundHandler)

	return apiServer
}

// Done returns a channel that will be closed when the server is done.
func (a *ApiServer) Done() <-chan bool {
	return a.done
}

// `helloHandler` responds with a simple greeting.
func (a *ApiServer) pingHandler(c fiber.Ctx) error {
	return c.SendString("pong")
}

func (a *ApiServer) startRecording(c fiber.Ctx) error {
	var req StartRecordingRequest

	err := json.Unmarshal(c.Body(), &req)

	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request payload")
	}

	opts := &pipeline.NewPipelineOptions{
		PageUrl:   req.Url,
		StreamUrl: req.StreamUrl,
	}

	if req.Chunking != (ChunkRequest{}) {
		opts.Chunking = &recorder.ChunkingOptions{
			ChunkDuration: req.Chunking.Duration,
		}
	}

	p, err := pipeline.NewPipeline(a.ctx, opts)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start recording pipeline")
	}

	if err := p.Start(); err != nil {
		log.Println("Error Occured Starting Pipeline", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start recording pipeline")
	}

	store.GetStore(&a.ctx).AddPipeLine(p.ID, p)

	return c.JSON(StartRecordingResponse{
		Status: "Recording Pipeline started",
		Id:     p.ID,
	})
}

func (a *ApiServer) stopRecording(c fiber.Ctx) error {
	log.Println("Stoppping Stream Called")
	var req StopRecordingRequest

	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request payload")
	}

	p, ok := store.GetStore(&a.ctx).GetPipeline(req.Id)

	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Pipeline not found")
	}

	if err := p.Stop(); err != nil {
		log.Println("Error Occured Stopping Pipeline", err)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to stop recording pipeline")
	}

	store.GetStore(&a.ctx).RemovePipeline(p.ID)

	return c.JSON(StopRecordingResponse{
		Status: "Recording stopped",
	})
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
			GracefulContext:       a.ctx,
			OnShutdownError: func(err error) {
				log.Printf("error shutting down the server: %v\n", err)
				close(a.done)
			},
			OnShutdownSuccess: func() {
				log.Println("server shutdown successfully")
				close(a.done)
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
