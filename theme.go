package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	"github.com/wailsapp/wails/v2/pkg/options"

	"zenith/internal/systheme"
)

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
