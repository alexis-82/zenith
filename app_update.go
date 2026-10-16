package main

import (
	"context"
	"errors"
	"fmt"
	goruntime "runtime"
	"sync"

	"zenith/internal/persist"
	ghupdate "zenith/internal/update"
)

// UpdateInfo is the result of an update check, as shown by the frontend.
type UpdateInfo struct {
	Current    string `json:"current"`
	Latest     string `json:"latest"`
	Available  bool   `json:"available"`  // latest is newer than current
	CanInstall bool   `json:"canInstall"` // there is a file to install on this OS
	NoRelease  bool   `json:"noRelease"`  // the repository has no release yet
	Notes      string `json:"notes"`
}

// updater holds the update state of the app. checker is replaceable in tests;
// install is the OS-specific installer launcher (update_<os>.go).
type updater struct {
	mu      sync.Mutex
	checker *ghupdate.Checker
	latest  *ghupdate.Release
	busy    bool
	install func(path string) error
	quit    func(ctx context.Context) // runtime.Quit; replaceable in tests
}

func (a *App) GetVersion() string {
	return version
}

// CheckUpdate asks GitHub for the latest release.
func (a *App) CheckUpdate() (UpdateInfo, error) {
	info := UpdateInfo{Current: version}
	r, err := a.updates.checker.Latest(a.ctx)
	if errors.Is(err, ghupdate.ErrNoRelease) {
		info.NoRelease = true
		return info, nil
	}
	if err != nil {
		return info, err
	}
	a.updates.mu.Lock()
	a.updates.latest = &r
	a.updates.mu.Unlock()

	info.Latest = r.Version
	info.Notes = r.Notes
	if newer, err := ghupdate.IsNewer(r.Version, version); err == nil {
		info.Available = newer // false for "dev" builds
	}
	_, assetErr := ghupdate.PickAsset(r.Assets, goruntime.GOOS, goruntime.GOARCH)
	info.CanInstall = info.Available && assetErr == nil
	return info, nil
}

// InstallUpdate downloads the release file for this OS, makes sure the data
// is saved, starts the installer and quits. It returns only on error.
func (a *App) InstallUpdate() error {
	u := &a.updates
	u.mu.Lock()
	if u.busy {
		u.mu.Unlock()
		return errors.New("aggiornamento già in corso")
	}
	r := u.latest
	u.busy = true
	u.mu.Unlock()
	defer func() {
		u.mu.Lock()
		u.busy = false
		u.mu.Unlock()
	}()

	if r == nil {
		return errors.New("controlla prima gli aggiornamenti")
	}
	if newer, err := ghupdate.IsNewer(r.Version, version); err != nil || !newer {
		return errors.New("nessun aggiornamento da installare")
	}
	asset, err := ghupdate.PickAsset(r.Assets, goruntime.GOOS, goruntime.GOARCH)
	if err != nil {
		return err
	}

	path, err := u.checker.Download(a.ctx, asset, func(done, total int64) {
		a.emit("update:progress", map[string]any{"done": done, "total": total})
	})
	if err != nil {
		return err
	}

	// The installer replaces the running program: save first, and do not
	// update while changes cannot be saved.
	if res, err := a.engine.PrepareClose(); res != persist.CloseOK {
		if res == persist.CloseConflict {
			return errors.New("c'è un conflitto con un altro dispositivo: risolvilo prima di aggiornare")
		}
		return fmt.Errorf("impossibile salvare le modifiche prima di aggiornare: %v", err)
	}

	if err := u.install(path); err != nil {
		return err
	}
	a.allowQuit.Store(true)
	u.quit(a.ctx)
	return nil
}
