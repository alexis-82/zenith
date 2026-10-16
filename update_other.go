//go:build !windows && !linux

package main

import "errors"

func installUpdate(path string) error {
	return errors.New("installazione automatica non disponibile su questo sistema: scarica la nuova versione da GitHub")
}
