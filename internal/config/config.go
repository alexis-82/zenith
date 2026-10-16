// Package config stores the per-computer Zenith settings (never shared via the
// data file).
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	MinAutosaveSeconds     = 1
	MaxAutosaveSeconds     = 300
	DefaultAutosaveSeconds = 2
)

type Settings struct {
	DataPath        string `json:"dataPath"`
	AutosaveSeconds int    `json:"autosaveSeconds"`
	Theme           string `json:"theme"` // system | light | dark
	CheckUpdates    bool   `json:"checkUpdates"`
}

func Defaults() Settings {
	return Settings{AutosaveSeconds: DefaultAutosaveSeconds, Theme: "system", CheckUpdates: true}
}

// Normalize clamps out-of-range values.
func (s Settings) Normalize() Settings {
	if s.AutosaveSeconds < MinAutosaveSeconds {
		s.AutosaveSeconds = MinAutosaveSeconds
	}
	if s.AutosaveSeconds > MaxAutosaveSeconds {
		s.AutosaveSeconds = MaxAutosaveSeconds
	}
	switch s.Theme {
	case "system", "light", "dark":
	default:
		s.Theme = "system"
	}
	return s
}

// Dir returns the per-user Zenith config directory.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Zenith"), nil
}

func file(dir string) string { return filepath.Join(dir, "settings.json") }

// Load reads the settings from dir; a missing or corrupt file yields defaults.
func Load(dir string) Settings {
	data, err := os.ReadFile(file(dir))
	if err != nil {
		return Defaults()
	}
	s := Defaults()
	if err := json.Unmarshal(data, &s); err != nil {
		return Defaults()
	}
	return s.Normalize()
}

// Save writes the settings to dir, creating it if needed.
func Save(dir string, s Settings) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Normalize(), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(file(dir), data, 0o644)
}
