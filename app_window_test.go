package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zenith/internal/config"
	"zenith/internal/systheme"
)

func TestWindowBackgroundFollowsTheme(t *testing.T) {
	system := bgLight
	if systheme.Dark() {
		system = bgDark
	}
	for theme, want := range map[string]any{"dark": bgDark, "light": bgLight, "system": system} {
		if got := *windowBackground(theme); got != want {
			t.Errorf("theme %q: background = %+v, want %+v", theme, got, want)
		}
	}
}

func TestThemedIndexWritesSavedTheme(t *testing.T) {
	const page = `<!DOCTYPE html><html lang="it"><head></head></html>`
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(page))
	})
	get := func(theme, path string) string {
		a := &App{settings: config.Settings{Theme: theme}}
		rec := httptest.NewRecorder()
		a.themedIndex(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec.Body.String()
	}

	if got := get("dark", "/"); !strings.Contains(got, `<html data-theme="dark" lang="it">`) {
		t.Errorf("dark: %s", got)
	}
	if got := get("light", "/index.html"); !strings.Contains(got, `<html data-theme="light" lang="it">`) {
		t.Errorf("light: %s", got)
	}
	// "system" is left to the page script; other files pass through untouched.
	if got := get("system", "/"); got != page {
		t.Errorf("system: %s", got)
	}
	if got := get("dark", "/assets/app.js"); got != page {
		t.Errorf("other path modified: %s", got)
	}
}
