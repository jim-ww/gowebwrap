// Package wailsrun launches the Wails webview for a resolved config. It has
// no Cobra/Viper imports so it can be driven by any front-end (CLI, tests,
// etc).
package wailsrun

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/jim-ww/gowebwrap/internal/config"
	"github.com/jim-ww/gowebwrap/internal/webkitcookies"
	"github.com/jim-ww/gowebwrap/internal/webkitua"
)

// Run opens the webview window for cfg and blocks until it's closed.
func Run(cfg config.Config) error {
	if cfg.DarkTheme && runtime.GOOS == "linux" {
		os.Setenv("GTK_THEME", "Adwaita:dark")
	}

	r, g, b, err := cfg.ParseBackgroundColor()
	if err != nil {
		return err
	}

	var iconBytes []byte
	if cfg.Icon != "" {
		iconBytes, err = os.ReadFile(cfg.Icon)
		if err != nil {
			return fmt.Errorf("reading icon %q: %w", cfg.Icon, err)
		}
	}

	gpuPolicy := application.WebviewGpuPolicyNever
	if cfg.GPUAcceleration {
		gpuPolicy = application.WebviewGpuPolicyAlways
	}

	app := application.New(application.Options{
		Name:        cfg.Name,
		Description: cfg.Description,
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            cfg.Name,
		URL:              cfg.URL,
		Width:            cfg.Width,
		Height:           cfg.Height,
		MaxWidth:         cfg.MaxWidth,
		MaxHeight:        cfg.MaxHeight,
		MinWidth:         cfg.MinWidth,
		MinHeight:        cfg.MinHeight,
		AlwaysOnTop:      cfg.AlwaysOnTop,
		DisableResize:    cfg.DisableResize,
		Frameless:        cfg.Frameless,
		StartState:       application.WindowStateNormal,
		BackgroundType:   application.BackgroundTypeSolid,
		BackgroundColour: application.NewRGB(r, g, b),
		DevToolsEnabled:  true,
		Linux: application.LinuxWindow{
			WebviewGpuPolicy: gpuPolicy,
			Icon:             iconBytes,
		},
	})

	if runtime.GOOS == "linux" {
		// The GTK3 backend only creates the native GtkWindow/WebKitWebView
		// (and fires off the first navigation) once app.Run() starts its
		// event loop, so there's no hook to set the UA before window
		// creation. Poll for the native handle from a separate goroutine
		// and force a reload once we can override it — the first request
		// may go out with the wrong UA and get rejected, but SetURL below
		// re-issues it correctly.
		go func() {
			for i := 0; i < 200; i++ {
				if nw := win.NativeWindow(); nw != nil {
					webkitcookies.EnablePersistentStorage(nw)
					if webkitua.SetUserAgent(nw, cfg.UserAgent) {
						win.SetURL(cfg.URL)
					}
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	return app.Run()
}
