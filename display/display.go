package display

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/OmGuptaIND/pkg"
	"github.com/chromedp/chromedp"
)

type DisplayOptions struct {
	ID string
	Wg *sync.WaitGroup

	Width  int
	Height int
	Depth  int
}

type Display struct {
	pulseSink string
	DisplayId string

	xvfb    *exec.Cmd
	browser *chromeDisplay

	*DisplayOptions
}

type chromeDisplay struct {
	chromeCtx    context.Context
	chromeCancel context.CancelFunc
}

// NewDisplay initializes a new Display with the specified options.
func NewDisplay(opts DisplayOptions) *Display {
	return &Display{
		DisplayId:      pkg.RandomDisplay(),
		pulseSink:      "",
		DisplayOptions: &opts,
	}
}

func (d *Display) GetDisplayId() string {
	return d.DisplayId
}

func (d *Display) GetWidth() int {
	return d.Width
}

func (d *Display) GetHeight() int {
	return d.Height
}

func (d *Display) GetSink() string {
	return d.pulseSink
}

func (d *Display) GetPulseMonitorId() string {
	return fmt.Sprintf("%s.monitor", d.ID)
}

// Launch starts the Xvfb server and Chrome with the specified URL.
func (d *Display) Launch(url string) error {
	if err := d.LaunchXvfb(); err != nil {
		return err
	}

	if _, err := d.LaunchChrome(url); err != nil {
		return err
	}

	log.Println("Chrome launched successfully")

	return nil
}

// Start launches the Xvfb server with the specified display.
func (d *Display) LaunchXvfb() error {
	if d.xvfb != nil {
		log.Println("Xvfb server is already running")
		return nil
	}

	log.Println("Starting Xvfb server...")

	dims := fmt.Sprintf("%dx%dx%d", d.Width, d.Height, d.Depth)
	xvfb := exec.Command("Xvfb", d.DisplayId, "-screen", "0", dims, "-ac", "-nolisten", "tcp")
	if err := xvfb.Start(); err != nil {
		return err
	}
	d.xvfb = xvfb

	log.Println("Xvfb server started")

	return nil
}

// Start a new Pulse Sink
func (d *Display) LaunchPulseSink() error {
	if d.pulseSink != "" {
		log.Println("Pulse Sink is already running")
		return nil
	}

	log.Println("Starting Pulse Sink...", d.ID)

	cmd := exec.Command("pactl",
		"load-module", "module-null-sink",
		fmt.Sprintf("sink_name=\"%s\"", d.ID),
		fmt.Sprintf("sink_properties=device.description=\"%s\"", d.ID),
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		log.Println("Failed to start Pulse Sink", err)
		return err
	}

	d.pulseSink = strings.TrimSpace(stdout.String())

	log.Println("Pulse Sink started", d.pulseSink)

	return nil
}

// LaunchChrome starts Chrome with the specified URL.
func (d *Display) LaunchChrome(url string) (*chromeDisplay, error) {
	log.Println("Launching Chrome...")
	if d.browser != nil {
		log.Println("Chrome is already running")
		return d.browser, nil
	}

	log.Println("Starting Chrome...", d.DisplayId)

	opts := []chromedp.ExecAllocatorOption{
		chromedp.ExecPath("chromium"),
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		chromedp.NoSandbox,

		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("excludeSwitches", "enable-automation"),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("enable-features", "NetworkService,NetworkServiceInProcess"),
		chromedp.Flag("disable-background-timer-throttling", true),
		chromedp.Flag("disable-backgrounding-occluded-windows", true),
		chromedp.Flag("disable-breakpad", true),
		chromedp.Flag("disable-client-side-phishing-detection", true),
		chromedp.Flag("disable-default-apps", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-features", "site-per-process,TranslateUI,BlinkGenPropertyTrees"),
		chromedp.Flag("disable-hang-monitor", true),
		chromedp.Flag("disable-ipc-flooding-protection", true),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("disable-prompt-on-repost", true),
		chromedp.Flag("disable-renderer-backgrounding", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("force-color-profile", "srgb"),
		chromedp.Flag("metrics-recording-only", true),
		chromedp.Flag("safebrowsing-disable-auto-update", true),
		chromedp.Flag("password-store", "basic"),
		chromedp.Flag("use-mock-keychain", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("allow-running-insecure-content", true),

		chromedp.Flag("kiosk", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("autoplay-policy", "no-user-gesture-required"),
		chromedp.Flag("window-position", "0,0"),
		chromedp.Flag("window-size", fmt.Sprintf("%d,%d", d.Width, d.Height)),
		chromedp.Flag("display", d.DisplayId),
		chromedp.Env(fmt.Sprintf("PULSE_SINK=%s", d.ID)),
	}

	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, cancel := chromedp.NewContext(allocCtx)

	err := chromedp.Run(ctx, chromedp.Navigate(url), chromedp.Evaluate(`window.screen.width`, &d.Width),
		chromedp.Evaluate(`window.screen.height`, &d.Height))

	if err != nil {
		log.Println("Failed to start Chrome", err)
		return nil, err
	}

	chromeDisplay := &chromeDisplay{
		chromeCtx:    ctx,
		chromeCancel: cancel,
	}

	d.browser = chromeDisplay

	go func() {
		<-chromeDisplay.chromeCtx.Done()
		log.Println("Chrome context done")
		cancelAlloc()
	}()

	log.Println("Chrome Launched")

	return chromeDisplay, nil
}

// Close stops the Chrome instance.
func (c *chromeDisplay) Close() {
	c.chromeCancel()
}

// Close stops the Chrome instance for the specified URL.
func (d *Display) CloseChrome(id string) {
	log.Println("Closing Chrome...")

	if d.browser != nil {
		d.browser.chromeCancel()
	}
}

// Close stops the Xvfb server and Chrome.
func (d *Display) Close() {
	log.Println("Closing display...")

	if d.browser != nil {
		d.browser.chromeCancel()
	}

	d.Wg.Add(2)
	go d.CloseXvfb()
	go d.ClosePulseSink()
}

// CloseXvfb stops the Xvfb server.
func (d *Display) CloseXvfb() {
	defer d.Wg.Done()
	log.Println("Closing Xvfb server...")

	if d.xvfb != nil {
		err := d.xvfb.Process.Signal(os.Interrupt)

		if err != nil {
			log.Println("Failed to stop Xvfb server")
		}

		err = d.xvfb.Wait()

		if err != nil {
			log.Println("Xvfb server exited with error", err)
		} else {
			log.Println("Xvfb server stopped")
		}

		d.xvfb = nil
	}
}

// ClosePulseSink stops the Pulse Sink.
func (d *Display) ClosePulseSink() {
	defer d.Wg.Done()
	log.Println("Closing Pulse Sink...")

	if d.pulseSink != "" {
		cmd := exec.Command("pactl", "unload-module", d.pulseSink)
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to stop Pulse Sink, Sink: %s\n", d.pulseSink)
		}

		log.Println("Pulse Sink stopped")

		d.pulseSink = ""
	}
}
