package display

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/chromedp/chromedp"
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

	chromeCtx    context.Context
	chromeCancel context.CancelFunc
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

	if err := d.LaunchChrome(url); err != nil {
		return err
	}

	log.Println("Chrome launched successfully")

	return nil
}

// Start launches the Xvfb server with the specified display.
func (d *Display) LaunchXvfb() error {
	log.Println("Starting Xvfb server...")

	dims := fmt.Sprintf("%dx%dx%d", d.opts.Width, d.opts.Height, d.opts.Depth)
	xvfb := exec.Command("Xvfb", d.opts.Display, "-screen", "0", dims, "-ac", "-nolisten", "tcp")
	if err := xvfb.Start(); err != nil {
		return err
	}
	d.xvfb = xvfb
	return nil
}

func (d *Display) LaunchChrome(url string) error {
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

	d.chromeCancel = cancel
	d.chromeCtx = ctx

	err := chromedp.Run(ctx, chromedp.Navigate(url), chromedp.Evaluate(`window.screen.width`, &d.opts.Width),
		chromedp.Evaluate(`window.screen.height`, &d.opts.Height))

	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// Close stops the Xvfb server and Chrome.
func (d *Display) Close() {
	log.Println("Closing display...")
	if d.chromeCancel != nil {
		d.chromeCancel()
	}

	if d.xvfb != nil {
		err := d.xvfb.Process.Signal(os.Interrupt)

		if err != nil {
			log.Println("Failed to stop Xvfb server")
		}

		d.xvfb = nil
	}
}

// TakeScreenshot captures a screenshot of the current display.
func (d *Display) TakeScreenshot() {
	var buf []byte
	if err := chromedp.Run(d.chromeCtx, chromedp.CaptureScreenshot(&buf)); err != nil {
		log.Fatal(err)
	}

	log.Println("Screenshot captured")

	if err := os.WriteFile("screenshot.png", buf, 0644); err != nil {
		log.Fatal(err)
	}
}
