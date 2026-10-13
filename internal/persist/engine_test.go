package persist

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"zenith/internal/model"
	"zenith/internal/store"
)

type recorder struct {
	mu       sync.Mutex
	statuses []Status
	external []bool
}

func (r *recorder) callbacks() Callbacks {
	return Callbacks{
		OnStatus: func(s Status, _ string) {
			r.mu.Lock()
			r.statuses = append(r.statuses, s)
			r.mu.Unlock()
		},
		OnExternalChange: func(c bool) {
			r.mu.Lock()
			r.external = append(r.external, c)
			r.mu.Unlock()
		},
	}
}

func (r *recorder) count(s Status) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, x := range r.statuses {
		if x == s {
			n++
		}
	}
	return n
}

func (r *recorder) externals() []bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]bool(nil), r.external...)
}

func (r *recorder) reset() {
	r.mu.Lock()
	r.statuses, r.external = nil, nil
	r.mu.Unlock()
}

// newFile creates a data file with one project and returns its path.
func newFile(t *testing.T, dir, name string) string {
	t.Helper()
	b := model.NewEmptyBoard()
	b.Projects = []model.Project{{ID: "p_1", Name: "Uno", Tasks: []model.Task{}}}
	p := filepath.Join(dir, name)
	if _, err := store.SaveAtomic(p, b); err != nil {
		t.Fatal(err)
	}
	return p
}

func renamed(b model.Board, name string) model.Board {
	b.Projects = append([]model.Project(nil), b.Projects...)
	b.Projects[0].Name = name
	return b
}

func loadName(t *testing.T, p string) string {
	t.Helper()
	b, _, err := store.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return b.Projects[0].Name
}

// externalWrite simulates another device rewriting the file.
func externalWrite(t *testing.T, p, name string) {
	t.Helper()
	b, _, err := store.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveAtomic(p, renamed(b, name)); err != nil {
		t.Fatal(err)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

func open(t *testing.T, interval time.Duration) (*Engine, *recorder, string, model.Board) {
	t.Helper()
	r := &recorder{}
	e := New(interval, r.callbacks())
	t.Cleanup(e.Close)
	p := newFile(t, t.TempDir(), "zenith.json")
	b, err := e.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	r.reset()
	return e, r, p, b
}

// --- Task 5: dirty, debounce, flush ---

func TestDebounceSingleWrite(t *testing.T) {
	e, r, p, b := open(t, 50*time.Millisecond)
	for i := 1; i <= 5; i++ {
		if err := e.UpdateBoard(int64(i), renamed(b, "v"+string(rune('0'+i)))); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, func() bool { return r.count(StatusSaved) == 1 })
	time.Sleep(150 * time.Millisecond)
	if n := r.count(StatusSaved); n != 1 {
		t.Fatalf("expected 1 write, got %d", n)
	}
	if got := loadName(t, p); got != "v5" {
		t.Fatalf("file has %q", got)
	}
	if r.count(StatusDirty) == 0 || r.count(StatusSaving) != 1 {
		t.Fatalf("unexpected statuses %v", r.statuses)
	}
}

func TestStaleSeqIgnored(t *testing.T) {
	e, _, p, b := open(t, time.Hour)
	e.UpdateBoard(2, renamed(b, "B"))
	e.UpdateBoard(1, renamed(b, "A"))
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := loadName(t, p); got != "B" {
		t.Fatalf("stale update applied: %q", got)
	}
}

func TestWriteErrorKeepsDirty(t *testing.T) {
	e, r, p, b := open(t, time.Hour)
	e.UpdateBoard(1, renamed(b, "X"))
	os.RemoveAll(filepath.Dir(p))
	if err := e.Flush(); err == nil {
		t.Fatal("expected write error")
	}
	if r.count(StatusError) != 1 || !e.dirty {
		t.Fatalf("expected error status and dirty, got %v dirty=%v", r.statuses, e.dirty)
	}
}

func TestInvalidBoardRejected(t *testing.T) {
	e, _, _, b := open(t, time.Hour)
	b.Columns = nil
	if err := e.UpdateBoard(1, b); err == nil {
		t.Fatal("expected validation error")
	}
}

// --- Task 6: conflicts and reload ---

func TestConflictOnFlush(t *testing.T) {
	e, r, p, b := open(t, time.Hour)
	e.Close() // no watcher: test the flush safety net alone
	e.UpdateBoard(1, renamed(b, "locale"))
	externalWrite(t, p, "esterno")

	if err := e.Flush(); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	if got := loadName(t, p); got != "esterno" {
		t.Fatalf("file overwritten: %q", got)
	}
	if ext := r.externals(); len(ext) != 1 || !ext[0] {
		t.Fatalf("expected conflict notification, got %v", ext)
	}

	// While paused, updates stay in memory only.
	e.UpdateBoard(2, renamed(b, "locale2"))
	e.Flush()
	if got := loadName(t, p); got != "esterno" {
		t.Fatalf("paused engine wrote: %q", got)
	}

	if _, err := e.ResolveConflict("keep"); err != nil {
		t.Fatal(err)
	}
	if got := loadName(t, p); got != "locale2" {
		t.Fatalf("keep did not overwrite: %q", got)
	}

	// Autosave resumes after resolution.
	e.SetInterval(30 * time.Millisecond)
	e.UpdateBoard(3, renamed(b, "dopo"))
	waitFor(t, func() bool { return loadName(t, p) == "dopo" })
}

func TestResolveReload(t *testing.T) {
	e, _, p, b := open(t, time.Hour)
	e.Close()
	e.UpdateBoard(1, renamed(b, "locale"))
	externalWrite(t, p, "esterno")
	e.Flush()

	got, err := e.ResolveConflict("reload")
	if err != nil {
		t.Fatal(err)
	}
	if got.Projects[0].Name != "esterno" || e.dirty || e.paused {
		t.Fatalf("reload failed: %+v dirty=%v paused=%v", got, e.dirty, e.paused)
	}
}

func TestResolveReloadMalformedKeepsConflict(t *testing.T) {
	e, _, p, b := open(t, time.Hour)
	e.Close()
	e.UpdateBoard(1, renamed(b, "locale"))
	os.WriteFile(p, []byte("{broken"), 0o644)
	e.Flush()
	if _, err := e.ResolveConflict("reload"); err == nil {
		t.Fatal("expected load error")
	}
	if !e.paused || !e.dirty {
		t.Fatal("conflict should stay open")
	}
	if _, err := e.ResolveConflict("keep"); err != nil {
		t.Fatal(err)
	}
	if got := loadName(t, p); got != "locale" {
		t.Fatalf("keep failed: %q", got)
	}
}

// --- Task 7: watcher ---

func TestWatcherExternalChange(t *testing.T) {
	e, r, p, b := open(t, time.Hour)

	// Our own write does not trigger a notification.
	e.UpdateBoard(1, renamed(b, "nostro"))
	if err := e.Flush(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * watchDebounce)
	if ext := r.externals(); len(ext) != 0 {
		t.Fatalf("own write reported as external: %v", ext)
	}

	// External write, nothing pending -> banner (conflict=false).
	externalWrite(t, p, "esterno")
	waitFor(t, func() bool { return len(r.externals()) == 1 })
	if r.externals()[0] {
		t.Fatal("expected conflict=false")
	}

	// External write with pending changes -> conflict.
	e.Reload()
	r.reset()
	e.UpdateBoard(2, renamed(b, "pendente"))
	externalWrite(t, p, "esterno2")
	waitFor(t, func() bool { return len(r.externals()) == 1 })
	if !r.externals()[0] || !e.paused {
		t.Fatal("expected conflict=true and paused")
	}
}

// --- Task 8: SetDataPath / CreateNewFile ---

func TestSetDataPathExisting(t *testing.T) {
	e, _, p, _ := open(t, time.Hour)
	other := newFile(t, filepath.Dir(p), "altro.json")
	externalWrite(t, other, "Altro")
	b, err := e.SetDataPath(other, false)
	if err != nil || b.Projects[0].Name != "Altro" || e.Path() != other {
		t.Fatalf("switch failed: %v %+v", err, b)
	}
}

func TestSetDataPathNewNeedsConfirm(t *testing.T) {
	e, _, p, b := open(t, time.Hour)
	e.UpdateBoard(1, renamed(b, "corrente"))
	target := filepath.Join(filepath.Dir(p), "nuovo.json")

	if _, err := e.SetDataPath(target, false); !errors.Is(err, ErrNeedsConfirm) {
		t.Fatalf("expected ErrNeedsConfirm, got %v", err)
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("file created without confirmation")
	}
	if _, err := e.SetDataPath(target, true); err != nil {
		t.Fatal(err)
	}
	if got := loadName(t, target); got != "corrente" {
		t.Fatalf("new file has %q", got)
	}
}

func TestSetDataPathNoBoardNeverCreates(t *testing.T) {
	e := New(time.Hour, Callbacks{})
	defer e.Close()
	target := filepath.Join(t.TempDir(), "zenith.json")
	for _, confirm := range []bool{false, true} {
		if _, err := e.SetDataPath(target, confirm); !errors.Is(err, ErrFileNotFound) {
			t.Fatalf("expected ErrFileNotFound, got %v", err)
		}
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("empty file created")
	}
}

func TestSetDataPathPendingFlushFailed(t *testing.T) {
	e, _, p, b := open(t, time.Hour)
	e.UpdateBoard(1, renamed(b, "pendente"))
	existing := newFile(t, t.TempDir(), "esistente.json")
	os.RemoveAll(filepath.Dir(p)) // old location unreachable

	if _, err := e.SetDataPath(existing, false); !errors.Is(err, ErrUnsavedChanges) {
		t.Fatalf("expected ErrUnsavedChanges, got %v", err)
	}
	if e.Path() != p {
		t.Fatal("engine switched despite unsaved changes")
	}

	target := filepath.Join(filepath.Dir(existing), "salvataggio.json")
	if _, err := e.SetDataPath(target, true); err != nil {
		t.Fatal(err)
	}
	if got := loadName(t, target); got != "pendente" {
		t.Fatalf("pending changes lost: %q", got)
	}
}

func TestSetDataPathConflictBlocksExisting(t *testing.T) {
	e, _, p, b := open(t, time.Hour)
	e.Close()
	e.UpdateBoard(1, renamed(b, "locale"))
	externalWrite(t, p, "esterno")
	e.Flush() // -> paused
	existing := newFile(t, t.TempDir(), "esistente.json")
	if _, err := e.SetDataPath(existing, false); !errors.Is(err, ErrUnsavedChanges) {
		t.Fatalf("expected ErrUnsavedChanges, got %v", err)
	}
}

func TestSetDataPathMalformedTarget(t *testing.T) {
	e, _, p, _ := open(t, time.Hour)
	bad := filepath.Join(filepath.Dir(p), "bad.json")
	os.WriteFile(bad, []byte("nope"), 0o644)
	if _, err := e.SetDataPath(bad, false); !errors.Is(err, store.ErrMalformed) {
		t.Fatalf("expected ErrMalformed, got %v", err)
	}
	if e.Path() != p {
		t.Fatal("engine left the old path")
	}
}

func TestCreateNewFile(t *testing.T) {
	e := New(time.Hour, Callbacks{})
	defer e.Close()
	dir := t.TempDir()
	target := filepath.Join(dir, "zenith.json")
	b, err := e.CreateNewFile(target)
	if err != nil || len(b.Columns) != 3 || len(b.Projects) != 0 {
		t.Fatalf("create failed: %v %+v", err, b)
	}

	existing := newFile(t, dir, "esistente.json")
	before, _ := os.ReadFile(existing)
	if _, err := e.CreateNewFile(existing); !errors.Is(err, ErrFileExists) {
		t.Fatalf("expected ErrFileExists, got %v", err)
	}
	after, _ := os.ReadFile(existing)
	if string(before) != string(after) {
		t.Fatal("existing file modified")
	}
}

func TestPrepareClose(t *testing.T) {
	e, _, p, b := open(t, time.Hour)
	if res, _ := e.PrepareClose(); res != CloseOK {
		t.Fatalf("clean engine: %v", res)
	}
	e.UpdateBoard(1, renamed(b, "uscita"))
	if res, err := e.PrepareClose(); res != CloseOK || err != nil {
		t.Fatalf("flush on close: %v %v", res, err)
	}
	if got := loadName(t, p); got != "uscita" {
		t.Fatalf("not saved on close: %q", got)
	}
	e.UpdateBoard(2, renamed(b, "fallito"))
	os.RemoveAll(filepath.Dir(p))
	if res, _ := e.PrepareClose(); res != CloseError {
		t.Fatalf("expected CloseError, got %v", res)
	}
}
