//go:build linux

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"

	ghupdate "zenith/internal/update"
)

// Zenith can be installed on Linux in three ways:
//   - .deb package (nfpm): /usr/bin/zenith, updated with apt;
//   - .rpm package (nfpm, Fedora): /usr/bin/zenith, updated with dnf;
//   - build/linux/install.sh: /usr/local/bin/zenith, updated with zenith-update.sh.
const (
	dpkgFileList      = "/var/lib/dpkg/info/zenith.list"
	debBinary         = "/usr/bin/zenith"
	rpmBinary         = "/usr/bin/zenith"
	linuxUpdateScript = "/usr/local/bin/zenith-update.sh"
)

// installedFromDeb reports whether the running binary belongs to the
// "zenith" Debian package.
func installedFromDeb() bool {
	if _, err := os.Stat(dpkgFileList); err != nil {
		return false
	}
	return runningExecutable() == debBinary
}

// installedFromRpm reports whether the running binary belongs to the
// "zenith" rpm package, asking the rpm database itself.
func installedFromRpm() bool {
	return ownedByRpm(runningExecutable(), func(path string) (string, error) {
		out, err := exec.Command("rpm", "-qf", "--qf", "%{NAME}", path).Output()
		return string(out), err
	})
}

// ownedByRpm is installedFromRpm with the rpm query passed in, for tests.
// No rpm, or a file owned by no package (rpm exits with 1), means false.
func ownedByRpm(exe string, query func(path string) (string, error)) bool {
	if exe != rpmBinary {
		return false
	}
	name, err := query(exe)
	return err == nil && strings.TrimSpace(name) == "zenith"
}

func runningExecutable() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}

// platformAsset picks the release file matching how Zenith was installed.
func platformAsset(assets []ghupdate.Asset) (ghupdate.Asset, error) {
	if installedFromDeb() {
		return ghupdate.PickDeb(assets, goruntime.GOARCH)
	}
	if installedFromRpm() {
		return ghupdate.PickRpm(assets, goruntime.GOARCH)
	}
	return ghupdate.PickAsset(assets, "linux", goruntime.GOARCH)
}

// installUpdate installs the downloaded file through pkexec (graphical
// password prompt) and starts the new version; the app quits right after.
func installUpdate(path string) error {
	pkexec, err := exec.LookPath("pkexec")
	if err != nil {
		return errors.New("pkexec non disponibile: aggiorna a mano da terminale (vedi README)")
	}

	var cmd *exec.Cmd
	if strings.HasSuffix(path, ".deb") {
		// apt also installs any new dependency of the package.
		cmd = exec.Command(pkexec, "env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "install", "-y", path)
	} else if strings.HasSuffix(path, ".rpm") {
		// Like apt, dnf also installs any new dependency of the package.
		dnf, err := exec.LookPath("dnf")
		if err != nil {
			return fmt.Errorf("dnf non disponibile: aggiorna a mano da terminale con: sudo dnf install %s", path)
		}
		cmd = exec.Command(pkexec, dnf, "install", "-y", path)
	} else {
		if _, err := os.Stat(linuxUpdateScript); err != nil {
			return fmt.Errorf("%s non trovato (Zenith non installato con build/linux/install.sh): aggiorna a mano da terminale con: sudo zenith-update.sh", linuxUpdateScript)
		}
		cmd = exec.Command(pkexec, linuxUpdateScript, path)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 126 {
			return errors.New("autorizzazione annullata: aggiornamento non installato")
		}
		return fmt.Errorf("installazione non riuscita: %v\n%s", err, strings.TrimSpace(string(out)))
	}
	os.RemoveAll(filepath.Dir(path))

	// The launcher sits next to the binary in both installation layouts.
	// Start it in its own session so it survives our exit.
	relaunch := exec.Command(filepath.Join(filepath.Dir(runningExecutable()), "zenith-launch"))
	relaunch.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := relaunch.Start(); err != nil {
		return fmt.Errorf("aggiornamento installato, ma impossibile riavviare Zenith: %w", err)
	}
	return nil
}
