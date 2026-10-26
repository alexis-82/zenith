package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	MinWindowWidth      = 900
	MinWindowHeight     = 600
	DefaultWindowWidth  = 1440
	DefaultWindowHeight = 860
)

// Window is the main window geometry of the last session on this computer.
// It lives in its own file so the frontend never round-trips it through
// SaveSettings.
type Window struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	Maximised bool `json:"maximised"`
}

func DefaultWindow() Window {
	return Window{Width: DefaultWindowWidth, Height: DefaultWindowHeight}
}

// Normalize replaces a missing size with the default and enforces the
// minimum size.
func (w Window) Normalize() Window {
	if w.Width <= 0 || w.Height <= 0 {
		w.Width, w.Height = DefaultWindowWidth, DefaultWindowHeight
	}
	w.Width = max(w.Width, MinWindowWidth)
	w.Height = max(w.Height, MinWindowHeight)
	return w
}

func windowFile(dir string) string { return filepath.Join(dir, "window.json") }

// LoadWindow reads the window geometry from dir; a missing or corrupt file
// yields the default.
func LoadWindow(dir string) Window {
	data, err := os.ReadFile(windowFile(dir))
	if err != nil {
		return DefaultWindow()
	}
	var w Window
	if err := json.Unmarshal(data, &w); err != nil {
		return DefaultWindow()
	}
	return w.Normalize()
}

// SaveWindow writes the window geometry to dir, creating it if needed.
func SaveWindow(dir string, w Window) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(w.Normalize(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(windowFile(dir), data, 0o644)
}
