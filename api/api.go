package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/OmGuptaIND/display"
	"github.com/OmGuptaIND/recorder"
	store "github.com/OmGuptaIND/store"
	"github.com/gofiber/fiber/v3"
)

// ApiServerOptions defines the configuration options for the ApiServer.
type ApiServerOptions struct {
	Port    int
	Wg      *sync.WaitGroup
	Ctx     context.Context
	Store   *store.Store
	Display *display.Display
}

// ApiServer represents an HTTP server that provides endpoints to manage media nodes.
type ApiServer struct {
	app  *fiber.App
	opts ApiServerOptions
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
	app.Post("/start-recording", apiServer.startRecording)
	app.Patch("/stop-recording", apiServer.stopRecording)
	app.Use(apiServer.notFoundHandler)

	return apiServer
}

// `helloHandler` responds with a simple greeting.
func (a *ApiServer) pingHandler(c fiber.Ctx) error {
	return c.SendString("pong")
}

func (a *ApiServer) startRecording(c fiber.Ctx) error {
	var req StartRecordingRequest

	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request payload")
	}

	if err := a.opts.Display.LaunchChrome(req.Url); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to launch Chrome")
	}

	rec := recorder.NewRecorder(recorder.NewRecorderOptions{
		Width:   1280,
		Height:  720,
		Depth:   24,
		Display: ":99",
	})

	if err := rec.StartRecording(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to start recording")
	}

	a.opts.Store.AddRecording(rec.ID, rec)

	return c.JSON(StartRecordingResponse{
		Status: "Recording started",
		Id:     rec.ID,
	})
}

func (a *ApiServer) stopRecording(c fiber.Ctx) error {
	var req StopRecordingRequest

	if err := json.Unmarshal(c.Body(), &req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request payload")
	}

	rec, ok := a.opts.Store.GetRecording(req.Id)

	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Recording not found")
	}

	if err := rec.StopRecording(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to stop recording")
	}

	a.opts.Store.RemoveRecording(req.Id)

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
			GracefulContext:       a.opts.Ctx,
			OnShutdownError: func(err error) {
				log.Printf("error shutting down the server: %v\n", err)
			},
			OnShutdownSuccess: func() {
				log.Println("server shutdown successfully")
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
