package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"zenith/internal/config"
	"zenith/internal/model"
	"zenith/internal/persist"
	"zenith/internal/store"
	ghupdate "zenith/internal/update"
)

// InitState is everything the frontend needs to render after startup or
// after switching data file.
type InitState struct {
	Settings  config.Settings `json:"settings"`
	Board     *model.Board    `json:"board"`
	LoadError string          `json:"loadError"`
}

type App struct {
	ctx       context.Context
	configDir string
	// mu guards settings and loadError: Wails runs every binding call in its
	// own goroutine, so e.g. SetDataPath and SaveSettings can overlap.
	mu        sync.Mutex
	settings  config.Settings
	loadError string
	engine    *persist.Engine
	allowQuit atomic.Bool
	// emitEvent is replaceable in tests (the Wails runtime needs a real app).
	emitEvent func(ctx context.Context, name string, data ...interface{})
	updates   updater
}

func NewApp() *App {
	return &App{
		emitEvent: runtime.EventsEmit,
		updates:   updater{checker: ghupdate.NewChecker(version), install: installUpdate, quit: runtime.Quit},
	}
}

func (a *App) emit(name string, data any) {
	a.emitEvent(a.ctx, name, data)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if dir, err := config.Dir(); err == nil {
		a.configDir = dir
		a.settings = config.Load(dir)
	} else {
		a.settings = config.Defaults()
	}
	a.engine = persist.New(a.interval(), persist.Callbacks{
		OnStatus: func(s persist.Status, errMsg string) {
			a.emit("save:status", map[string]any{"status": s, "error": errMsg})
		},
		OnExternalChange: func(conflict bool) {
			a.emit("file:external-change", map[string]any{"conflict": conflict})
		},
	})
	if a.settings.DataPath != "" {
		if _, err := a.engine.Open(a.settings.DataPath); err != nil {
			a.loadError = loadErrorMessage(a.settings.DataPath, err)
		}
	}
}

// beforeClose returns true to prevent the window from closing.
func (a *App) beforeClose(ctx context.Context) bool {
	if a.allowQuit.Load() {
		a.engine.Close()
		return false
	}
	res, err := a.engine.PrepareClose()
	if res == persist.CloseOK {
		a.engine.Close()
		return false
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	a.emit("app:close-blocked", map[string]any{"reason": res, "error": msg})
	return true
}

func (a *App) interval() time.Duration {
	return time.Duration(a.settings.AutosaveSeconds) * time.Second
}

func (a *App) state(b *model.Board) InitState {
	return InitState{Settings: a.settings, Board: b, LoadError: a.loadError}
}

// Init returns the startup state.
func (a *App) Init() InitState {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state(a.engine.Board())
}

// UpdateBoard receives the frontend working state; seq must increase.
func (a *App) UpdateBoard(seq int64, b model.Board) error {
	return coded(a.engine.UpdateBoard(seq, b))
}

func (a *App) SaveNow() error {
	return coded(a.engine.Flush())
}

func (a *App) Reload() (model.Board, error) {
	b, err := a.engine.Reload()
	return b, coded(err)
}

func (a *App) ResolveConflict(choice string) (*model.Board, error) {
	b, err := a.engine.ResolveConflict(choice)
	return b, coded(err)
}

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

// Quit closes the app. force=true exits without saving; otherwise pending
// changes are saved first and, if that fails, app:close-blocked is emitted
// again. The check is done here rather than relying on runtime.Quit
// triggering OnBeforeClose.
func (a *App) Quit(force bool) {
	if !force && a.beforeClose(a.ctx) {
		return
	}
	a.allowQuit.Store(true)
	runtime.Quit(a.ctx)
}

// coded prefixes known errors with a stable code the frontend can match.
func coded(err error) error {
	if err == nil {
		return nil
	}
	codes := []struct {
		target error
		code   string
	}{
		{persist.ErrNeedsConfirm, "ERR_NEEDS_CONFIRM"},
		{persist.ErrFileNotFound, "ERR_FILE_NOT_FOUND"},
		{persist.ErrUnsavedChanges, "ERR_UNSAVED_CHANGES"},
		{persist.ErrFileExists, "ERR_FILE_EXISTS"},
		{persist.ErrConflict, "ERR_CONFLICT"},
		{persist.ErrNoFile, "ERR_NO_FILE"},
		{persist.ErrIsDir, "ERR_IS_DIR"},
		{store.ErrMalformed, "ERR_MALFORMED"},
		{store.ErrInvalid, "ERR_INVALID"},
		{store.ErrNotFound, "ERR_FILE_NOT_FOUND"},
	}
	for _, c := range codes {
		if errors.Is(err, c.target) {
			return fmt.Errorf("%s: %s", c.code, err.Error())
		}
	}
	return err
}

func loadErrorMessage(path string, err error) string {
	switch {
	case errors.Is(err, store.ErrNotFound):
		return fmt.Sprintf("Il file dati %q non esiste. Se Nextcloud sta ancora sincronizzando attendi e riavvia, oppure scegli un altro file nelle Impostazioni.", path)
	case errors.Is(err, store.ErrMalformed), errors.Is(err, store.ErrInvalid):
		return fmt.Sprintf("Il file dati %q non è leggibile (%v). Il file non è stato modificato.", path, err)
	default:
		return fmt.Sprintf("Impossibile aprire %q: %v", path, err)
	}
}
