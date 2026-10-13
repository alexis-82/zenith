// Package persist owns the data file: delayed autosave, external change
// detection, conflicts and data file switching. It knows nothing about Wails;
// the app wires the callbacks to runtime events.
package persist

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"zenith/internal/model"
	"zenith/internal/store"
)

type Status string

const (
	StatusSaved  Status = "saved"
	StatusDirty  Status = "dirty"
	StatusSaving Status = "saving"
	StatusError  Status = "error"
	StatusPaused Status = "paused" // conflict open, autosave suspended
)

var (
	ErrNoFile         = errors.New("nessun file dati configurato")
	ErrConflict       = errors.New("il file è stato modificato da un altro dispositivo")
	ErrFileNotFound   = errors.New("il file non esiste")
	ErrNeedsConfirm   = errors.New("il file non esiste: serve conferma per salvarci i dati attuali")
	ErrUnsavedChanges = errors.New("ci sono modifiche non salvate")
	ErrFileExists     = errors.New("il file esiste già")
	ErrIsDir          = errors.New("il percorso indicato è una cartella")
)

// Callbacks are invoked with the engine lock held: they must not call back
// into the engine.
type Callbacks struct {
	OnStatus         func(status Status, errMsg string)
	OnExternalChange func(conflict bool)
}

// CloseResult tells the app whether the window may close.
type CloseResult string

const (
	CloseOK       CloseResult = "ok"
	CloseConflict CloseResult = "conflict"
	CloseError    CloseResult = "error"
)

type Engine struct {
	mu       sync.Mutex
	cb       Callbacks
	interval time.Duration

	path         string
	board        *model.Board // nil when no data file is loaded
	dirty        bool
	paused       bool
	lastSeq      int64
	lastHash     string // hash of the bytes last loaded or written by us
	notifiedHash string // last external hash already reported
	timer        *time.Timer
	watcher      *watcher
}

func New(interval time.Duration, cb Callbacks) *Engine {
	if cb.OnStatus == nil {
		cb.OnStatus = func(Status, string) {}
	}
	if cb.OnExternalChange == nil {
		cb.OnExternalChange = func(bool) {}
	}
	return &Engine{interval: interval, cb: cb}
}

// Open loads path and makes it the active data file. On error the engine is
// left without a data file (autosave disabled).
func (e *Engine) Open(path string) (model.Board, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	b, h, err := store.Load(path)
	if err != nil {
		e.closeFileLocked()
		return model.Board{}, err
	}
	e.switchToLocked(path, b, h)
	return b, nil
}

// Path returns the active data file path ("" if none).
func (e *Engine) Path() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.path
}

// Board returns a copy of the in-memory board, or nil if no file is loaded.
func (e *Engine) Board() *model.Board {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.board == nil {
		return nil
	}
	b := *e.board
	return &b
}

func (e *Engine) SetInterval(d time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.interval = d
}

// UpdateBoard receives the working state from the frontend. Calls with a seq
// not greater than the last one are stale (out-of-order IPC) and ignored.
func (e *Engine) UpdateBoard(seq int64, b model.Board) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if seq <= e.lastSeq {
		return nil
	}
	e.lastSeq = seq
	if e.board == nil {
		return ErrNoFile
	}
	if err := b.Validate(); err != nil {
		return err
	}
	e.board = &b
	e.dirty = true
	if e.paused {
		e.cb.OnStatus(StatusPaused, "")
		return nil
	}
	e.cb.OnStatus(StatusDirty, "")
	e.stopTimerLocked()
	e.timer = time.AfterFunc(e.interval, func() { e.Flush() })
	return nil
}

// Flush writes pending changes now.
func (e *Engine) Flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.flushLocked()
}

func (e *Engine) flushLocked() error {
	if !e.dirty || e.paused || e.board == nil {
		return nil
	}
	// Safety net for missed watcher events: never overwrite a file that
	// changed since we last read or wrote it.
	if h, err := store.HashFile(e.path); err == nil && h != e.lastHash {
		e.enterConflictLocked(h)
		return ErrConflict
	}
	e.cb.OnStatus(StatusSaving, "")
	h, err := store.SaveAtomic(e.path, *e.board)
	if err != nil {
		e.cb.OnStatus(StatusError, err.Error())
		return err
	}
	e.lastHash = h
	e.dirty = false
	e.cb.OnStatus(StatusSaved, "")
	return nil
}

func (e *Engine) enterConflictLocked(diskHash string) {
	e.paused = true
	e.notifiedHash = diskHash
	e.stopTimerLocked()
	e.cb.OnStatus(StatusPaused, "")
	e.cb.OnExternalChange(true)
}

// fileChanged is called by the watcher (debounced) when the data file's
// directory reports activity on the data file.
func (e *Engine) fileChanged() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.board == nil || e.paused {
		return
	}
	h, err := store.HashFile(e.path)
	if err != nil || h == e.lastHash || h == e.notifiedHash {
		return
	}
	if e.dirty {
		e.enterConflictLocked(h)
		return
	}
	e.notifiedHash = h
	e.cb.OnExternalChange(false)
}

// Reload replaces the in-memory board with the file on disk, discarding
// pending changes. On error nothing changes.
func (e *Engine) Reload() (model.Board, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.reloadLocked()
}

func (e *Engine) reloadLocked() (model.Board, error) {
	if e.board == nil {
		return model.Board{}, ErrNoFile
	}
	b, h, err := store.Load(e.path)
	if err != nil {
		return model.Board{}, err
	}
	e.stopTimerLocked()
	e.board = &b
	e.lastHash = h
	e.notifiedHash = ""
	e.dirty = false
	e.paused = false
	e.cb.OnStatus(StatusSaved, "")
	return b, nil
}

// ResolveConflict: "reload" discards local changes and returns the disk board
// (on load error the conflict stays open); "keep" overwrites the file with the
// local state.
func (e *Engine) ResolveConflict(choice string) (*model.Board, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	switch choice {
	case "reload":
		b, err := e.reloadLocked()
		if err != nil {
			return nil, err
		}
		return &b, nil
	case "keep":
		if e.board == nil {
			return nil, ErrNoFile
		}
		if h, err := store.HashFile(e.path); err == nil {
			e.lastHash = h
		}
		e.paused = false
		e.notifiedHash = ""
		e.dirty = true
		return nil, e.flushLocked()
	default:
		return nil, fmt.Errorf("scelta non valida: %q", choice)
	}
}

// SetDataPath switches the active data file. See plan.md "Cambio percorso".
func (e *Engine) SetDataPath(p string, confirmWrite bool) (model.Board, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	exists, err := pathExists(p)
	if err != nil {
		return model.Board{}, err
	}
	pendingUnsaved := e.savePendingLocked()

	if exists {
		if pendingUnsaved {
			return model.Board{}, ErrUnsavedChanges
		}
		b, h, err := store.Load(p)
		if err != nil {
			return model.Board{}, err
		}
		e.switchToLocked(p, b, h)
		return b, nil
	}

	if e.board == nil {
		return model.Board{}, ErrFileNotFound
	}
	if !confirmWrite {
		return model.Board{}, ErrNeedsConfirm
	}
	h, err := store.SaveAtomic(p, *e.board)
	if err != nil {
		return model.Board{}, err
	}
	b := *e.board
	e.switchToLocked(p, b, h)
	return b, nil
}

// CreateNewFile creates an empty data file at p (which must not exist) and
// activates it. It is the only way to create an empty board.
func (e *Engine) CreateNewFile(p string) (model.Board, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	exists, err := pathExists(p)
	if err != nil {
		return model.Board{}, err
	}
	if exists {
		return model.Board{}, ErrFileExists
	}
	if e.savePendingLocked() {
		return model.Board{}, ErrUnsavedChanges
	}
	b := model.NewEmptyBoard()
	h, err := store.SaveAtomic(p, b)
	if err != nil {
		return model.Board{}, err
	}
	e.switchToLocked(p, b, h)
	return b, nil
}

// savePendingLocked tries to flush pending changes to the current file and
// reports whether some changes remain unsaved (flush failed or conflict open).
func (e *Engine) savePendingLocked() bool {
	if !e.dirty {
		return false
	}
	if e.paused {
		return true
	}
	return e.flushLocked() != nil
}

// PrepareClose flushes pending changes and tells whether the app may exit.
func (e *Engine) PrepareClose() (CloseResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.dirty {
		return CloseOK, nil
	}
	if e.paused {
		return CloseConflict, nil
	}
	if err := e.flushLocked(); err != nil {
		if errors.Is(err, ErrConflict) {
			return CloseConflict, nil
		}
		return CloseError, err
	}
	return CloseOK, nil
}

// Close stops the timer and the watcher.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.stopTimerLocked()
	e.stopWatcherLocked()
}

func (e *Engine) switchToLocked(path string, b model.Board, hash string) {
	e.stopTimerLocked()
	e.stopWatcherLocked()
	e.path = path
	e.board = &b
	e.lastHash = hash
	e.notifiedHash = ""
	e.dirty = false
	e.paused = false
	if w, err := startWatcher(path, e.fileChanged); err == nil {
		e.watcher = w
	}
	// Without a watcher the hash check in flushLocked still prevents
	// overwriting external changes.
	e.cb.OnStatus(StatusSaved, "")
}

func (e *Engine) closeFileLocked() {
	e.stopTimerLocked()
	e.stopWatcherLocked()
	e.path = ""
	e.board = nil
	e.dirty = false
	e.paused = false
}

func (e *Engine) stopTimerLocked() {
	if e.timer != nil {
		e.timer.Stop()
		e.timer = nil
	}
}

func (e *Engine) stopWatcherLocked() {
	if e.watcher != nil {
		e.watcher.close()
		e.watcher = nil
	}
}

func pathExists(p string) (bool, error) {
	info, err := os.Stat(p)
	if err == nil {
		if info.IsDir() {
			return false, ErrIsDir
		}
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}
