//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Installed by build/linux/install.sh.
const (
	linuxUpdateScript = "/usr/local/bin/zenith-update.sh"
	linuxLauncher     = "/usr/local/bin/zenith-launch.sh"
)

// installUpdate installs the downloaded tarball through pkexec (graphical
// password prompt) and starts the new version; the app quits right after.
func installUpdate(path string) error {
	manual := "aggiorna a mano da terminale con: sudo zenith-update.sh"
	if _, err := os.Stat(linuxUpdateScript); err != nil {
		return fmt.Errorf("%s non trovato (Zenith non installato con build/linux/install.sh): %s", linuxUpdateScript, manual)
	}
	pkexec, err := exec.LookPath("pkexec")
	if err != nil {
		return errors.New("pkexec non disponibile: " + manual)
	}
	out, err := exec.Command(pkexec, linuxUpdateScript, path).CombinedOutput()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && (exit.ExitCode() == 126 || exit.ExitCode() == 127) {
			return errors.New("autorizzazione annullata: aggiornamento non installato")
		}
		return fmt.Errorf("installazione non riuscita: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	os.RemoveAll(filepath.Dir(path))

	// Start the new version in its own session so it survives our exit.
	cmd := exec.Command(linuxLauncher)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("aggiornamento installato, ma impossibile riavviare Zenith: %w", err)
	}
	return nil
}
