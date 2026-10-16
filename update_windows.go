//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	goruntime "runtime"

	"golang.org/x/sys/windows"

	ghupdate "zenith/internal/update"
)

// installUpdate starts the downloaded setup.exe; the app quits right after.
// ShellExecute (unlike exec.Command) also handles an installer that asks
// for elevation.
func installUpdate(path string) error {
	err := windows.ShellExecute(0, windows.StringToUTF16Ptr("open"), windows.StringToUTF16Ptr(path),
		nil, windows.StringToUTF16Ptr(filepath.Dir(path)), windows.SW_SHOWNORMAL)
	if err != nil {
		return fmt.Errorf("impossibile avviare il setup: %w", err)
	}
	return nil
}

func platformAsset(assets []ghupdate.Asset) (ghupdate.Asset, error) {
	return ghupdate.PickAsset(assets, "windows", goruntime.GOARCH)
}
