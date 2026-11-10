package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"zenith/internal/config"
	"zenith/internal/systheme"
)

// readWindow returns the current window geometry. A minimised window has no
// meaningful size, and a maximised one only records the flag.
func readWindow(ctx context.Context) (config.Window, bool) {
	if runtime.WindowIsMinimised(ctx) {
		return config.Window{}, false
	}
	if runtime.WindowIsMaximised(ctx) {
		return config.Window{Maximised: true}, true
	}
	w, h := runtime.WindowGetSize(ctx)
	return config.Window{Width: w, Height: h}, true
}

// rememberWindow stores the window geometry for the next launch on this
// computer. A maximised window keeps the last normal size, so that
// un-maximising it after the next launch restores that size. Failures are
// ignored: the next launch simply uses the previous (or default) size.
func (a *App) rememberWindow() {
	if a.readWindow == nil || a.configDir == "" {
		return
	}
	w, ok := a.readWindow(a.ctx)
	if !ok {
		return
	}
	if w.Maximised {
		prev := config.LoadWindow(a.configDir)
		w.Width, w.Height = prev.Width, prev.Height
	}
	_ = config.SaveWindow(a.configDir, w)
}

// Colours of --bg in frontend/src/style.css. The window shows them before
// the page paints, so they must match or a flash of the wrong colour shows
// on start; afterwards frontend/src/lib/theme.ts keeps them in sync.
var (
	bgLight = options.RGBA{R: 0xf6, G: 0xf7, B: 0xf9, A: 255}
	bgDark  = options.RGBA{R: 0x15, G: 0x18, B: 0x1d, A: 255}
)

// windowBackground is the window background before the page is loaded.
func windowBackground(theme string) *options.RGBA {
	if theme == "dark" || (theme == "system" && systheme.Dark()) {
		return &bgDark
	}
	return &bgLight
}

// themedIndex is an AssetServer middleware that writes the saved theme into
// <html data-theme> of index.html, so the page starts with the right colours
// instead of turning dark only after Init. With the "system" theme it writes
// nothing: the script at the top of index.html works it out.
func (a *App) themedIndex(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || (r.URL.Path != "" && r.URL.Path != "/" && r.URL.Path != "/index.html") {
			next.ServeHTTP(w, r)
			return
		}
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		body := rec.Body.Bytes()
		if theme := a.GetSettings().Theme; rec.Code == http.StatusOK && (theme == "dark" || theme == "light") {
			body = bytes.Replace(body, []byte("<html "), []byte(`<html data-theme="`+theme+`" `), 1)
		}
		for k, v := range rec.Header() {
			w.Header()[k] = v
		}
		w.Header().Del("Content-Length")
		w.WriteHeader(rec.Code)
		w.Write(body)
	})
}
