package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var jsonFilter = []runtime.FileFilter{{DisplayName: "File JSON (*.json)", Pattern: "*.json"}}

func (a *App) PickExistingFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Scegli il file dati di Zenith",
		Filters: jsonFilter,
	})
}

func (a *App) PickNewFile() (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Crea un nuovo file dati",
		DefaultFilename: "zenith.json",
		Filters:         jsonFilter,
	})
}

// OpenURL opens a Markdown link in the system browser; only http, https and
// mailto are allowed.
func (a *App) OpenURL(u string) error {
	if !isSafeURL(u) {
		return fmt.Errorf("link non consentito: %s", u)
	}
	runtime.BrowserOpenURL(a.ctx, u)
	return nil
}

func isSafeURL(u string) bool {
	parsed, err := url.Parse(strings.TrimSpace(u))
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Host != ""
	case "mailto":
		return true
	}
	return false
}
