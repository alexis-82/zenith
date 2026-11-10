package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"zenith/internal/config"
	"zenith/internal/model"
)

func (a *App) GetSettings() config.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

// SaveSettings updates autosave interval and theme (the data path changes
// only through SetDataPath / CreateNewFile).
func (a *App) SaveSettings(s config.Settings) (config.Settings, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	s.DataPath = a.settings.DataPath
	s = s.Normalize()
	if err := a.persistSettings(s); err != nil {
		return a.settings, err
	}
	a.engine.SetInterval(a.interval())
	return a.settings, nil
}

func (a *App) SetDataPath(p string, confirmWrite bool) (InitState, error) {
	p, err := cleanPath(p)
	if err != nil {
		return InitState{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	b, err := a.engine.SetDataPath(p, confirmWrite)
	if err != nil {
		return InitState{}, coded(err)
	}
	return a.activated(p, b)
}

func (a *App) CreateNewFile(p string) (InitState, error) {
	p, err := cleanPath(p)
	if err != nil {
		return InitState{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	b, err := a.engine.CreateNewFile(p)
	if err != nil {
		return InitState{}, coded(err)
	}
	return a.activated(p, b)
}

// cleanPath requires an absolute path: a relative one would be resolved
// against the process working directory, which changes between launches.
func cleanPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("percorso vuoto")
	}
	if !filepath.IsAbs(p) {
		return "", errors.New("indica il percorso completo del file, non un percorso relativo")
	}
	return filepath.Clean(p), nil
}

// activated must be called with a.mu held.
func (a *App) activated(p string, b model.Board) (InitState, error) {
	a.loadError = ""
	s := a.settings
	s.DataPath = p
	if err := a.persistSettings(s); err != nil {
		return a.state(&b), fmt.Errorf("file attivato, ma impossibile salvare le impostazioni: %w", err)
	}
	return a.state(&b), nil
}

func (a *App) persistSettings(s config.Settings) error {
	if a.configDir == "" {
		return errors.New("cartella di configurazione non disponibile")
	}
	if err := config.Save(a.configDir, s); err != nil {
		return err
	}
	a.settings = s
	return nil
}
