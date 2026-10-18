//go:build linux

package main

import (
	"errors"
	"testing"
)

func TestOwnedByRpm(t *testing.T) {
	owner := func(name string, err error) func(string) (string, error) {
		return func(string) (string, error) { return name, err }
	}

	if !ownedByRpm("/usr/bin/zenith", owner("zenith", nil)) {
		t.Fatal("binary of the zenith package: expected true")
	}
	if ownedByRpm("/usr/bin/zenith", owner("", errors.New(`exec: "rpm": executable file not found in $PATH`))) {
		t.Fatal("rpm not installed: expected false")
	}
	if ownedByRpm("/usr/bin/zenith", owner("file /usr/bin/zenith is not owned by any package", errors.New("exit status 1"))) {
		t.Fatal("file owned by no package: expected false")
	}
	if ownedByRpm("/usr/bin/zenith", owner("other", nil)) {
		t.Fatal("binary of another package: expected false")
	}
	if ownedByRpm("/usr/bin/zenith", owner("zenith", errors.New("exit status 1"))) {
		t.Fatal("failed rpm query: expected false even with plausible output")
	}

	called := false
	query := func(string) (string, error) { called = true; return "zenith", nil }
	if ownedByRpm("/usr/local/bin/zenith", query) || called {
		t.Fatal("install.sh layout: expected false without querying rpm")
	}
}
