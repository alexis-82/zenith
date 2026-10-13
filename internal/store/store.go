// Package store reads and atomically writes the Zenith data file.
package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"zenith/internal/model"
)

var (
	// ErrNotFound: the data file does not exist.
	ErrNotFound = errors.New("file non trovato")
	// ErrMalformed: the file is not valid JSON.
	ErrMalformed = errors.New("file JSON malformato")
	// ErrInvalid: the JSON is well formed but violates the board invariants.
	ErrInvalid = errors.New("file dati non valido")
)

// Hash returns the hex SHA-256 of data.
func Hash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// HashFile returns the hash of the file content.
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", ErrNotFound
		}
		return "", err
	}
	return Hash(data), nil
}

// Load reads, parses and validates the data file. It returns the board and
// the hash of the bytes read.
func Load(path string) (model.Board, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return model.Board{}, "", ErrNotFound
		}
		return model.Board{}, "", err
	}
	var b model.Board
	if err := json.Unmarshal(data, &b); err != nil {
		return model.Board{}, "", fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	if err := b.Validate(); err != nil {
		return model.Board{}, "", fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	normalize(&b)
	return b, Hash(data), nil
}

// normalize replaces nil slices so the frontend always gets arrays.
func normalize(b *model.Board) {
	if b.Projects == nil {
		b.Projects = []model.Project{}
	}
	for i := range b.Projects {
		if b.Projects[i].Tasks == nil {
			b.Projects[i].Tasks = []model.Task{}
		}
	}
}

// SaveAtomic writes the board to a temp file in the same directory and then
// renames it over path. It returns the hash of the written bytes.
func SaveAtomic(path string, b model.Board) (string, error) {
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(filepath.Dir(path), ".zenith-*.tmp")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", err
	}
	ok = true
	return Hash(data), nil
}
