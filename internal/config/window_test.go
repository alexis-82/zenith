package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWindowDefaultWhenMissingOrCorrupt(t *testing.T) {
	dir := t.TempDir()
	if got := LoadWindow(dir); got != DefaultWindow() {
		t.Fatalf("expected default, got %+v", got)
	}
	os.WriteFile(filepath.Join(dir, "window.json"), []byte("{not json"), 0o644)
	if got := LoadWindow(dir); got != DefaultWindow() {
		t.Fatalf("expected default for corrupt file, got %+v", got)
	}
}

func TestWindowRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Zenith")
	want := Window{Width: 1920, Height: 1080, Maximised: true}
	if err := SaveWindow(dir, want); err != nil {
		t.Fatal(err)
	}
	if got := LoadWindow(dir); got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestWindowNormalize(t *testing.T) {
	cases := []struct{ in, want Window }{
		{Window{}, DefaultWindow()},
		{Window{Width: 1000, Height: -1}, DefaultWindow()},
		{Window{Width: 400, Height: 300}, Window{Width: MinWindowWidth, Height: MinWindowHeight}},
		{Window{Width: 1280, Height: 720, Maximised: true}, Window{Width: 1280, Height: 720, Maximised: true}},
	}
	for _, c := range cases {
		if got := c.in.Normalize(); got != c.want {
			t.Errorf("Normalize(%+v) = %+v, want %+v", c.in, got, c.want)
		}
	}
}
