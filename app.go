package main

import (
	"context"
	"errors"
	"fmt"
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
	// readWindow is replaceable in tests, like emitEvent; ok=false skips saving.
	readWindow func(ctx context.Context) (w config.Window, ok bool)
	updates    updater
}

func NewApp() *App {
	return &App{
		emitEvent:  runtime.EventsEmit,
		readWindow: readWindow,
		updates:    updater{checker: ghupdate.NewChecker(version), install: ghupdate.Install, quit: runtime.Quit},
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
		a.rememberWindow()
		a.engine.Close()
		return false
	}
	res, err := a.engine.PrepareClose()
	if res == persist.CloseOK {
		a.rememberWindow()
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
