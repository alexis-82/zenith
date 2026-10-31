//go:build linux

package main

import (
	"errors"
	"testing"
)

func TestOwnedByPackage(t *testing.T) {
	owner := func(name string, err error) func(string) (string, error) {
		return func(string) (string, error) { return name, err }
	}

	if !ownedByPackage("/usr/bin/zenith", "/usr/bin/zenith", owner("zenith\n", nil)) {
		t.Fatal("binary of the zenith package: expected true")
	}
	if ownedByPackage("/usr/bin/zenith", "/usr/bin/zenith", owner("", errors.New(`exec: "pacman": executable file not found in $PATH`))) {
		t.Fatal("package manager not installed: expected false")
	}
	if ownedByPackage("/usr/bin/zenith", "/usr/bin/zenith", owner("", errors.New("exit status 1"))) {
		t.Fatal("file owned by no package: expected false")
	}
	if ownedByPackage("/usr/bin/zenith", "/usr/bin/zenith", owner("other", nil)) {
		t.Fatal("binary of another package: expected false")
	}
	if ownedByPackage("/usr/bin/zenith", "/usr/bin/zenith", owner("zenith", errors.New("exit status 1"))) {
		t.Fatal("failed query: expected false even with plausible output")
	}

	called := false
	query := func(string) (string, error) { called = true; return "zenith", nil }
	if ownedByPackage("/usr/local/bin/zenith", "/usr/bin/zenith", query) || called {
		t.Fatal("install.sh layout: expected false without querying the package manager")
	}
}
