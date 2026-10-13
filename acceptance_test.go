package main

// Acceptance tests at the App (Wails binding) level: they drive the same
// methods the frontend calls and check the data file and settings on disk,
// including "restart" (a new App reading the same config dir).

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"zenith/internal/model"
	"zenith/internal/store"
)

type event struct {
	name string
	data map[string]any
}

type testApp struct {
	*App
	mu     sync.Mutex
	events []event
}

func (ta *testApp) eventsNamed(name string) []map[string]any {
	ta.mu.Lock()
	defer ta.mu.Unlock()
	var out []map[string]any
	for _, e := range ta.events {
		if e.name == name {
			out = append(out, e.data)
		}
	}
	return out
}

// useConfigHome points os.UserConfigDir to dir on every OS.
func useConfigHome(t *testing.T, dir string) {
	t.Setenv("AppData", dir)         // Windows
	t.Setenv("XDG_CONFIG_HOME", dir) // Linux
	t.Setenv("HOME", dir)            // macOS: $HOME/Library/Application Support
	os.MkdirAll(filepath.Join(dir, "Library", "Application Support"), 0o755)
}

// startApp simulates an app launch with the config in configHome.
func startApp(t *testing.T, configHome string) *testApp {
	t.Helper()
	useConfigHome(t, configHome)
	ta := &testApp{App: NewApp()}
	ta.emitEvent = func(_ context.Context, name string, data ...interface{}) {
		ta.mu.Lock()
		defer ta.mu.Unlock()
		m, _ := data[0].(map[string]any)
		ta.events = append(ta.events, event{name, m})
	}
	ta.startup(context.Background())
	t.Cleanup(ta.engine.Close)
	return ta
}

var seq int64

func update(t *testing.T, a *testApp, b model.Board) {
	t.Helper()
	seq++
	if err := a.UpdateBoard(seq, b); err != nil {
		t.Fatal(err)
	}
}

func withProject(b model.Board, name string, tasks ...model.Task) model.Board {
	b.Projects = append(append([]model.Project(nil), b.Projects...), model.Project{ID: model.NewID("p"), Name: name, Tasks: tasks})
	return b
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

func readBoard(t *testing.T, p string) model.Board {
	t.Helper()
	b, _, err := store.Load(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// newDataFile creates a data file through the app, like "Crea nuovo file…".
func newDataFile(t *testing.T, a *testApp) (string, model.Board) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "Nextcloud", "zenith.json")
	os.MkdirAll(filepath.Dir(p), 0o755)
	s, err := a.CreateNewFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return p, *s.Board
}

func TestAcceptance_FirstLaunchIsEmpty(t *testing.T) {
	home := t.TempDir()
	a := startApp(t, home)
	s := a.Init()
	if s.Board != nil || s.LoadError != "" || s.Settings.DataPath != "" {
		t.Fatalf("expected virgin state, got %+v", s)
	}
	if s.Settings.AutosaveSeconds != 2 || s.Settings.Theme != "system" {
		t.Fatalf("unexpected default settings %+v", s.Settings)
	}
}

func TestAcceptance_CreateNewFileAndRestart(t *testing.T) {
	home := t.TempDir()
	a := startApp(t, home)
	p, b := newDataFile(t, a)

	onDisk := readBoard(t, p)
	names := []string{}
	for _, c := range onDisk.Columns {
		names = append(names, c.Name)
	}
	if strings.Join(names, ",") != "Implementare,In corso,Terminato" || len(onDisk.Projects) != 0 {
		t.Fatalf("unexpected new file: %+v", onDisk)
	}
	if b.Columns[0].ID != onDisk.Columns[0].ID {
		t.Fatal("returned board differs from file")
	}

	a.engine.Close()
	restarted := startApp(t, home)
	s := restarted.Init()
	if s.Settings.DataPath != p || s.Board == nil || s.Board.Columns[0].ID != b.Columns[0].ID {
		t.Fatalf("file not reloaded after restart: %+v", s)
	}
}

func TestAcceptance_ChangesSurviveRestart(t *testing.T) {
	home := t.TempDir()
	a := startApp(t, home)
	p, b := newDataFile(t, a)
	a.SaveSettings(a.GetSettings()) // keep defaults, 2s
	c := b.Columns

	// project + global column added, renamed and moved between "In corso" and "Terminato"
	b = withProject(b, "Test2",
		model.Task{ID: "t_a", Title: "Task", Description: "**md**", ColumnID: c[1].ID, Order: 0},
		model.Task{ID: "t_b", Title: "Sopra", ColumnID: c[1].ID, Order: 1},
	)
	b.Columns = []model.Column{c[0], c[1], {ID: "c_rev", Name: "Revisione"}, c[2]}
	update(t, a, b)
	if err := a.SaveNow(); err != nil {
		t.Fatal(err)
	}

	a.engine.Close()
	got := startApp(t, home).Init().Board
	if got == nil || len(got.Projects) != 1 || got.Projects[0].Name != "Test2" {
		t.Fatalf("project not persisted: %+v", got)
	}
	if got.Columns[2].Name != "Revisione" || got.Columns[3].ID != c[2].ID {
		t.Fatalf("column order not persisted: %+v", got.Columns)
	}
	tasks := got.Projects[0].Tasks
	if tasks[0].ColumnID != c[1].ID || tasks[0].Description != "**md**" || tasks[1].Order != 1 {
		t.Fatalf("tasks not persisted: %+v", tasks)
	}
	_ = p
}

func TestAcceptance_AutosaveAfterDelay(t *testing.T) {
	a := startApp(t, t.TempDir())
	p, b := newDataFile(t, a)
	s := a.GetSettings()
	s.AutosaveSeconds = 1
	if _, err := a.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(p)

	// a burst of changes...
	for i := 0; i < 3; i++ {
		update(t, a, withProject(b, "P"+string(rune('0'+i))))
		time.Sleep(100 * time.Millisecond)
	}
	// ...does not touch the file immediately...
	if now, _ := os.ReadFile(p); !bytes.Equal(now, before) {
		t.Fatal("file written before the autosave delay")
	}
	// ...and is written once, about 1s after the last change.
	waitUntil(t, 3*time.Second, func() bool { return len(readBoard(t, p).Projects) == 1 })
	if readBoard(t, p).Projects[0].Name != "P2" {
		t.Fatal("last change not saved")
	}
	saved := 0
	for _, e := range a.eventsNamed("save:status") {
		if fmt.Sprint(e["status"]) == "saved" {
			saved++
		}
	}
	// 1 from CreateNewFile + 1 autosave
	if saved != 2 {
		t.Fatalf("expected a single autosave write, got %d saved events", saved-1)
	}
}

func TestAcceptance_SaveOnClose(t *testing.T) {
	a := startApp(t, t.TempDir())
	p, b := newDataFile(t, a)
	s := a.GetSettings()
	s.AutosaveSeconds = 300
	a.SaveSettings(s)

	update(t, a, withProject(b, "Uscita"))
	if prevent := a.beforeClose(context.Background()); prevent {
		t.Fatal("close prevented although save succeeded")
	}
	if got := readBoard(t, p); len(got.Projects) != 1 || got.Projects[0].Name != "Uscita" {
		t.Fatalf("change not saved on close: %+v", got)
	}
}

func TestAcceptance_SaveOnCloseFailureBlocksClose(t *testing.T) {
	a := startApp(t, t.TempDir())
	p, b := newDataFile(t, a)
	s := a.GetSettings()
	s.AutosaveSeconds = 300
	a.SaveSettings(s)

	update(t, a, withProject(b, "Persa?"))
	os.RemoveAll(filepath.Dir(p))
	if prevent := a.beforeClose(context.Background()); !prevent {
		t.Fatal("close allowed although save failed")
	}
	ev := a.eventsNamed("app:close-blocked")
	if len(ev) != 1 || fmt.Sprint(ev[0]["reason"]) != "error" {
		t.Fatalf("expected close-blocked error event, got %v", ev)
	}
}

func TestAcceptance_ExternalChange(t *testing.T) {
	a := startApp(t, t.TempDir())
	p, b := newDataFile(t, a)

	// own save: no external-change event
	update(t, a, withProject(b, "Nostro"))
	a.SaveNow()
	time.Sleep(time.Second)
	if ev := a.eventsNamed("file:external-change"); len(ev) != 0 {
		t.Fatalf("own save reported as external: %v", ev)
	}

	// another device writes, nothing pending: banner, reload shows new data
	ext := readBoard(t, p)
	ext.Projects[0].Name = "Da altro PC"
	store.SaveAtomic(p, ext)
	waitUntil(t, 3*time.Second, func() bool { return len(a.eventsNamed("file:external-change")) == 1 })
	if a.eventsNamed("file:external-change")[0]["conflict"] != false {
		t.Fatal("expected conflict=false")
	}
	got, err := a.Reload()
	if err != nil || got.Projects[0].Name != "Da altro PC" {
		t.Fatalf("reload failed: %v %+v", err, got)
	}

	// pending local change + external write: conflict, autosave paused
	s := a.GetSettings()
	s.AutosaveSeconds = 1
	a.SaveSettings(s)
	local := got
	local.Projects = []model.Project{{ID: got.Projects[0].ID, Name: "Locale", Tasks: []model.Task{}}}
	update(t, a, local)
	ext.Projects[0].Name = "Esterno 2"
	store.SaveAtomic(p, ext)
	waitUntil(t, 3*time.Second, func() bool { return len(a.eventsNamed("file:external-change")) == 2 })
	if a.eventsNamed("file:external-change")[1]["conflict"] != true {
		t.Fatal("expected conflict=true")
	}
	time.Sleep(1500 * time.Millisecond) // autosave interval elapsed
	if readBoard(t, p).Projects[0].Name != "Esterno 2" {
		t.Fatal("autosave wrote during conflict")
	}

	// "Mantieni le mie"
	if _, err := a.ResolveConflict("keep"); err != nil {
		t.Fatal(err)
	}
	if readBoard(t, p).Projects[0].Name != "Locale" {
		t.Fatal("keep did not overwrite")
	}
}

func TestAcceptance_ConflictReload(t *testing.T) {
	a := startApp(t, t.TempDir())
	p, b := newDataFile(t, a)
	b = withProject(b, "Base")
	update(t, a, b)
	a.SaveNow()

	local := withProject(b, "Locale")
	update(t, a, local)
	ext := readBoard(t, p)
	ext.Projects[0].Name = "Esterno"
	store.SaveAtomic(p, ext)
	waitUntil(t, 3*time.Second, func() bool { return len(a.eventsNamed("file:external-change")) == 1 })

	got, err := a.ResolveConflict("reload")
	if err != nil || len(got.Projects) != 1 || got.Projects[0].Name != "Esterno" {
		t.Fatalf("reload choice failed: %v %+v", err, got)
	}
}

func TestAcceptance_ChangeDataPath(t *testing.T) {
	a := startApp(t, t.TempDir())
	p, b := newDataFile(t, a)
	update(t, a, withProject(b, "Miei dati"))

	// to a non-existing file: confirmation required, then current data written there
	target := filepath.Join(filepath.Dir(p), "spostato.json")
	if _, err := a.SetDataPath(target, false); err == nil || !strings.HasPrefix(err.Error(), "ERR_NEEDS_CONFIRM") {
		t.Fatalf("expected ERR_NEEDS_CONFIRM, got %v", err)
	}
	s, err := a.SetDataPath(target, true)
	if err != nil || s.Settings.DataPath != target {
		t.Fatalf("move failed: %v %+v", err, s)
	}
	if readBoard(t, target).Projects[0].Name != "Miei dati" {
		t.Fatal("data not written to new path")
	}

	// to an existing file: it is loaded
	other := filepath.Join(filepath.Dir(p), "altro.json")
	store.SaveAtomic(other, withProject(model.NewEmptyBoard(), "Altro"))
	s, err = a.SetDataPath(other, false)
	if err != nil || s.Board.Projects[0].Name != "Altro" || a.GetSettings().DataPath != other {
		t.Fatalf("load existing failed: %v %+v", err, s)
	}

	// to a malformed file: error, path unchanged, file untouched
	bad := filepath.Join(filepath.Dir(p), "bad.json")
	os.WriteFile(bad, []byte("{oops"), 0o644)
	if _, err := a.SetDataPath(bad, false); err == nil || !strings.HasPrefix(err.Error(), "ERR_MALFORMED") {
		t.Fatalf("expected ERR_MALFORMED, got %v", err)
	}
	if a.GetSettings().DataPath != other {
		t.Fatal("path changed despite error")
	}
	if data, _ := os.ReadFile(bad); string(data) != "{oops" {
		t.Fatal("malformed file modified")
	}
}

func TestAcceptance_VirginAppNeverCreatesEmptyFile(t *testing.T) {
	a := startApp(t, t.TempDir())
	target := filepath.Join(t.TempDir(), "zenith.json")
	for _, confirm := range []bool{false, true} {
		if _, err := a.SetDataPath(target, confirm); err == nil || !strings.HasPrefix(err.Error(), "ERR_FILE_NOT_FOUND") {
			t.Fatalf("expected ERR_FILE_NOT_FOUND, got %v", err)
		}
	}
	if _, err := os.Stat(target); err == nil {
		t.Fatal("empty file created")
	}
}

func TestAcceptance_MissingOrMalformedFileAtStartup(t *testing.T) {
	home := t.TempDir()
	a := startApp(t, home)
	p, _ := newDataFile(t, a)
	a.engine.Close()

	os.WriteFile(p, []byte("{broken"), 0o644)
	b := startApp(t, home)
	s := b.Init()
	if s.Board != nil || !strings.Contains(s.LoadError, "non è leggibile") {
		t.Fatalf("expected load error, got %+v", s)
	}
	if err := b.UpdateBoard(1_000_000, model.NewEmptyBoard()); err == nil {
		t.Fatal("updates accepted without a loaded file")
	}
	if data, _ := os.ReadFile(p); string(data) != "{broken" {
		t.Fatal("malformed file overwritten")
	}
	b.engine.Close()

	os.Remove(p)
	c := startApp(t, home)
	if s := c.Init(); s.Board != nil || !strings.Contains(s.LoadError, "Nextcloud") {
		t.Fatalf("expected missing-file error, got %+v", s)
	}
	if _, err := os.Stat(p); err == nil {
		t.Fatal("missing file was recreated")
	}
}

func TestAcceptance_SettingsPersist(t *testing.T) {
	home := t.TempDir()
	a := startApp(t, home)
	s := a.GetSettings()
	s.Theme = "dark"
	s.AutosaveSeconds = 0 // out of range -> clamped
	got, err := a.SaveSettings(s)
	if err != nil || got.AutosaveSeconds != 1 {
		t.Fatalf("save settings: %v %+v", err, got)
	}
	a.engine.Close()
	if s := startApp(t, home).GetSettings(); s.Theme != "dark" || s.AutosaveSeconds != 1 {
		t.Fatalf("settings not persisted: %+v", s)
	}
}

func TestAcceptance_SameFileFromTwoComputers(t *testing.T) {
	pc1 := startApp(t, t.TempDir())
	p, b := newDataFile(t, pc1)
	update(t, pc1, withProject(b, "Condiviso", model.Task{ID: "t_1", Title: "X", ColumnID: b.Columns[2].ID}))
	pc1.SaveNow()

	pc2 := startApp(t, t.TempDir()) // different computer = different config dir
	s, err := pc2.SetDataPath(p, false)
	if err != nil {
		t.Fatal(err)
	}
	b1 := readBoard(t, p)
	if s.Board.Projects[0].Name != "Condiviso" || s.Board.Projects[0].Tasks[0].ColumnID != b1.Columns[2].ID {
		t.Fatalf("second computer sees different data: %+v", s.Board)
	}
}

func TestAcceptance_RelativePathRejected(t *testing.T) {
	a := startApp(t, t.TempDir())
	for _, p := range []string{"zenith.json", `Nextcloud\zenith.json`, "./zenith.json"} {
		if _, err := a.SetDataPath(p, true); err == nil || !strings.Contains(err.Error(), "percorso completo") {
			t.Fatalf("SetDataPath(%q): expected relative path error, got %v", p, err)
		}
		if _, err := a.CreateNewFile(p); err == nil || !strings.Contains(err.Error(), "percorso completo") {
			t.Fatalf("CreateNewFile(%q): expected relative path error, got %v", p, err)
		}
	}
	if _, err := os.Stat("zenith.json"); err == nil {
		t.Fatal("file created in the working directory")
	}
}
