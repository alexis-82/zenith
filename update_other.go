//go:build !windows && !linux

package main

import (
	"errors"
	goruntime "runtime"

	ghupdate "zenith/internal/update"
)

func installUpdate(path string) error {
	return errors.New("installazione automatica non disponibile su questo sistema: scarica la nuova versione da GitHub")
}

func platformAsset(assets []ghupdate.Asset) (ghupdate.Asset, error) {
	return ghupdate.PickAsset(assets, goruntime.GOOS, goruntime.GOARCH)
}
