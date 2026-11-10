//go:build !windows && !linux

package update

import (
	"errors"
	goruntime "runtime"
)

func Install(path string) error {
	return errors.New("installazione automatica non disponibile su questo sistema: scarica la nuova versione da GitHub")
}

func PlatformAsset(assets []Asset) (Asset, error) {
	return PickAsset(assets, goruntime.GOOS, goruntime.GOARCH)
}
