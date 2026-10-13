// Package model defines the shared Zenith data file format.
package model

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// CurrentVersion is the only data file version this build understands.
const CurrentVersion = 1

// Board is the whole content of the shared JSON data file.
type Board struct {
	Version  int       `json:"version"`
	Columns  []Column  `json:"columns"`
	Projects []Project `json:"projects"`
}

// Column is a global board column; its position is its index in Board.Columns.
type Column struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Project struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Tasks []Task `json:"tasks"`
}

type Task struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ColumnID    string `json:"columnId"`
	Order       int    `json:"order"`
}

// DefaultColumnNames are the columns of a freshly created data file.
var DefaultColumnNames = []string{"Implementare", "In corso", "Terminato"}

// NewID returns prefix + "_" + 12 random hex chars.
// The frontend generator (frontend/src/lib/ids.ts) uses the same format.
func NewID(prefix string) string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b)
}

// NewEmptyBoard returns a board with the default columns and no projects.
func NewEmptyBoard() Board {
	b := Board{Version: CurrentVersion, Columns: []Column{}, Projects: []Project{}}
	for _, name := range DefaultColumnNames {
		b.Columns = append(b.Columns, Column{ID: NewID("c"), Name: name})
	}
	return b
}

// Validate checks the structural invariants of a board.
func (b Board) Validate() error {
	if b.Version != CurrentVersion {
		return fmt.Errorf("versione del file non supportata: %d", b.Version)
	}
	if len(b.Columns) == 0 {
		return errors.New("il file non contiene colonne")
	}
	ids := map[string]bool{}
	use := func(id, what string) error {
		if id == "" {
			return fmt.Errorf("%s senza id", what)
		}
		if ids[id] {
			return fmt.Errorf("id duplicato: %s", id)
		}
		ids[id] = true
		return nil
	}
	columns := map[string]bool{}
	for _, c := range b.Columns {
		if err := use(c.ID, "colonna"); err != nil {
			return err
		}
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("colonna %s senza nome", c.ID)
		}
		columns[c.ID] = true
	}
	for _, p := range b.Projects {
		if err := use(p.ID, "progetto"); err != nil {
			return err
		}
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("progetto %s senza nome", p.ID)
		}
		for _, t := range p.Tasks {
			if err := use(t.ID, "task"); err != nil {
				return err
			}
			if strings.TrimSpace(t.Title) == "" {
				return fmt.Errorf("task %s senza titolo", t.ID)
			}
			if !columns[t.ColumnID] {
				return fmt.Errorf("task %s in una colonna inesistente: %s", t.ID, t.ColumnID)
			}
		}
	}
	return nil
}
