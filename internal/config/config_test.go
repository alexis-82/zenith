package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsWhenMissing(t *testing.T) {
	if got := Load(t.TempDir()); got != Defaults() {
		t.Fatalf("expected defaults, got %+v", got)
	}
	if d := Defaults(); d.DataPath != "" || d.AutosaveSeconds != 2 || d.Theme != "system" {
		t.Fatalf("unexpected defaults %+v", d)
	}
}

func TestRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Zenith")
	want := Settings{DataPath: `C:\Nextcloud\zenith.json`, AutosaveSeconds: 10, Theme: "dark"}
	if err := Save(dir, want); err != nil {
		t.Fatal(err)
	}
	if got := Load(dir); got != want {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want Settings }{
		{Settings{AutosaveSeconds: 0, Theme: "light"}, Settings{AutosaveSeconds: 1, Theme: "light"}},
		{Settings{AutosaveSeconds: 1000, Theme: "dark"}, Settings{AutosaveSeconds: 300, Theme: "dark"}},
		{Settings{AutosaveSeconds: 5, Theme: "pink"}, Settings{AutosaveSeconds: 5, Theme: "system"}},
	}
	for _, c := range cases {
		if got := c.in.Normalize(); got != c.want {
			t.Errorf("Normalize(%+v) = %+v, want %+v", c.in, got, c.want)
		}
	}
}

func TestCorruptFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{not json"), 0o644)
	if got := Load(dir); got != Defaults() {
		t.Fatalf("expected defaults, got %+v", got)
	}
}

func TestCheckUpdatesDefaultForOlderFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"dataPath":"x","autosaveSeconds":3,"theme":"dark"}`), 0o644)
	if s := Load(dir); !s.CheckUpdates {
		t.Fatal("settings saved before the update feature must default to checking updates")
	}
	if err := Save(dir, Settings{AutosaveSeconds: 2, Theme: "system", CheckUpdates: false}); err != nil {
		t.Fatal(err)
	}
	if Load(dir).CheckUpdates {
		t.Fatal("disabled update check not persisted")
	}
}
