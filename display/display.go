package display

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

type DisplayOptions struct {
	Width  int
	Height int
	Depth  int

	Display string `yaml:"-"`
}

type Display struct {
	xvfb *exec.Cmd
	opts DisplayOptions

	mu       sync.RWMutex
	browsers map[string]*chromeDisplay
}

type chromeDisplay struct {
	id           string
	chromeCtx    context.Context
	chromeCancel context.CancelFunc

	DisplayOptions
}

// NewDisplay initializes a new Display with the specified options.
func NewDisplay(opts DisplayOptions) *Display {
	return &Display{
		opts: opts,
	}
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

	dims := fmt.Sprintf("%dx%dx%d", d.opts.Width, d.opts.Height, d.opts.Depth)
	xvfb := exec.Command("Xvfb", d.opts.Display, "-screen", "0", dims, "-ac", "-nolisten", "tcp")
	if err := xvfb.Start(); err != nil {
		return err
	}
	d.xvfb = xvfb
	return nil
}

// LaunchChrome starts Chrome with the specified URL.
func (d *Display) LaunchChrome(url string) (*chromeDisplay, error) {
	log.Println("Launching Chrome...")
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
		chromedp.Flag("window-size", fmt.Sprintf("%d,%d", d.opts.Width, d.opts.Height)),
		chromedp.Flag("display", d.opts.Display),
	}

	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	ctx, cancel := chromedp.NewContext(allocCtx)

	err := chromedp.Run(ctx, chromedp.Navigate(url), chromedp.Evaluate(`window.screen.width`, &d.opts.Width),
		chromedp.Evaluate(`window.screen.height`, &d.opts.Height))

	if err != nil {
		log.Println(err)
		return nil, err
	}

	chromeDisplay := &chromeDisplay{
		id:             uuid.New().String(),
		chromeCtx:      ctx,
		chromeCancel:   cancel,
		DisplayOptions: d.opts,
	}

	d.mu.Lock()
	d.browsers[chromeDisplay.id] = chromeDisplay
	d.mu.Unlock()

	go func() {
		<-chromeDisplay.chromeCtx.Done()
		log.Println("Chrome exited")
		d.mu.Lock()
		delete(d.browsers, chromeDisplay.id)
		d.mu.Unlock()
	}()

	return chromeDisplay, nil
}

// Close stops the Chrome instance.
func (c *chromeDisplay) Close() {
	c.chromeCancel()
}

// Close stops the Chrome instance for the specified URL.
func (d *Display) CloseChrome(id string) bool {
	log.Println("Closing Chrome...")
	d.mu.RLock()
	defer d.mu.RUnlock()

	if browser, ok := d.browsers[id]; ok {
		browser.chromeCancel()
		delete(d.browsers, id)
		return true
	}

	return false
}

// Close stops the Xvfb server and Chrome.
func (d *Display) Close() {
	log.Println("Closing display...")
	d.mu.RLock()
	defer d.mu.RUnlock()

	for _, browser := range d.browsers {
		browser.chromeCancel()
	}

	if d.xvfb != nil {
		err := d.xvfb.Process.Signal(os.Interrupt)

		if err != nil {
			log.Println("Failed to stop Xvfb server")
		}

		d.xvfb = nil
	}
}
