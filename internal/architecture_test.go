package internal_test

import (
	"go/build"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

const module = "zenith"

// forbidden are the imports internal packages must not use: the Wails runtime
// stays in the App facade, and internal/ must not depend on package main.
var forbidden = map[string]string{
	"github.com/wailsapp/wails/v2/pkg/runtime": "Wails runtime",
	module: "package main",
}

// TestInternalPackagesStayWailsFree checks the imports of every package under
// internal/, following imports between module packages, for Linux and Windows.
func TestInternalPackagesStayWailsFree(t *testing.T) {
	// Every directory under internal/, nested ones included.
	var dirs []string
	err := filepath.WalkDir(".", func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() && path != "." {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, goos := range []string{"linux", "windows"} {
		ctx := build.Default
		ctx.GOOS = goos
		seen := map[string]bool{} // each package is checked once per GOOS
		for _, dir := range dirs {
			pkg, err := ctx.ImportDir(dir, 0)
			if err != nil {
				continue // not a Go package (or no files for this GOOS)
			}
			path := module + "/internal/" + filepath.ToSlash(dir)
			if seen[path] {
				continue
			}
			seen[path] = true
			checkImports(t, &ctx, goos, path, pkg.Imports, seen)
		}
	}
}

func checkImports(t *testing.T, ctx *build.Context, goos, from string, imports []string, seen map[string]bool) {
	t.Helper()
	for _, imp := range imports {
		if what, bad := forbidden[imp]; bad {
			t.Errorf("%s (%s) imports %s (%s)", from, goos, imp, what)
		}
		if seen[imp] || !strings.HasPrefix(imp, module+"/internal/") {
			continue
		}
		seen[imp] = true
		dir := filepath.FromSlash(strings.TrimPrefix(imp, module+"/internal/"))
		pkg, err := ctx.ImportDir(dir, 0)
		if err != nil {
			t.Errorf("%s: %v", imp, err)
			continue
		}
		checkImports(t, ctx, goos, imp, pkg.Imports, seen)
	}
}
