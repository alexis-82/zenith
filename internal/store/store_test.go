package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"zenith/internal/model"
)

func sampleBoard() model.Board {
	b := model.NewEmptyBoard()
	b.Projects = []model.Project{{
		ID:   "p_1",
		Name: "Progetto",
		Tasks: []model.Task{
			{ID: "t_1", Title: "Uno", Description: "**md**", ColumnID: b.Columns[1].ID, Order: 0},
		},
	}}
	return b
}

func tmpFiles(t *testing.T, dir string) []string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(dir, ".zenith-*.tmp"))
	return m
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zenith.json")
	b := sampleBoard()

	hash, err := SaveAtomic(path, b)
	if err != nil {
		t.Fatal(err)
	}
	got, loadHash, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, b) {
		t.Fatalf("round trip mismatch:\n%+v\n%+v", got, b)
	}
	fileHash, _ := HashFile(path)
	if hash != fileHash || loadHash != fileHash {
		t.Fatalf("hash mismatch: save=%s load=%s file=%s", hash, loadHash, fileHash)
	}
	if left := tmpFiles(t, dir); len(left) != 0 {
		t.Fatalf("temp files left: %v", left)
	}
}

func TestOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zenith.json")
	b := sampleBoard()
	if _, err := SaveAtomic(path, b); err != nil {
		t.Fatal(err)
	}
	b.Projects[0].Name = "Rinominato"
	if _, err := SaveAtomic(path, b); err != nil {
		t.Fatal(err)
	}
	got, _, err := Load(path)
	if err != nil || got.Projects[0].Name != "Rinominato" {
		t.Fatalf("overwrite failed: %v %+v", err, got)
	}
}

func TestLoadErrors(t *testing.T) {
	dir := t.TempDir()

	if _, _, err := Load(filepath.Join(dir, "missing.json")); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	malformed := filepath.Join(dir, "bad.json")
	content := []byte(`{"version": 1, "columns": [`)
	os.WriteFile(malformed, content, 0o644)
	if _, _, err := Load(malformed); !errors.Is(err, ErrMalformed) {
		t.Fatalf("expected ErrMalformed, got %v", err)
	}
	after, _ := os.ReadFile(malformed)
	if !bytes.Equal(after, content) {
		t.Fatalf("malformed file was modified")
	}

	invalid := filepath.Join(dir, "invalid.json")
	os.WriteFile(invalid, []byte(`{"version": 1, "columns": [], "projects": []}`), 0o644)
	if _, _, err := Load(invalid); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestLoadNormalizesNilSlices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "z.json")
	os.WriteFile(path, []byte(`{"version":1,"columns":[{"id":"c_1","name":"A"}],"projects":[{"id":"p_1","name":"P"}]}`), 0o644)
	b, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if b.Projects[0].Tasks == nil {
		t.Fatal("tasks should be an empty slice")
	}
}

func TestSaveFailureLeavesNoTemp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing-dir", "zenith.json")
	if _, err := SaveAtomic(path, sampleBoard()); err == nil {
		t.Fatal("expected error for missing directory")
	}
	if left := tmpFiles(t, dir); len(left) != 0 {
		t.Fatalf("temp files left: %v", left)
	}
}
