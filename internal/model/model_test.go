package model

import (
	"regexp"
	"testing"
)

func validBoard() Board {
	b := NewEmptyBoard()
	b.Projects = []Project{{
		ID:    "p_1",
		Name:  "Progetto",
		Tasks: []Task{{ID: "t_1", Title: "Task", ColumnID: b.Columns[0].ID}},
	}}
	return b
}

func TestNewEmptyBoard(t *testing.T) {
	b := NewEmptyBoard()
	if err := b.Validate(); err != nil {
		t.Fatalf("empty board invalid: %v", err)
	}
	if len(b.Columns) != 3 || b.Columns[0].Name != "Implementare" || b.Columns[2].Name != "Terminato" {
		t.Fatalf("unexpected default columns: %+v", b.Columns)
	}
	if len(b.Projects) != 0 {
		t.Fatalf("expected no projects")
	}
	if !regexp.MustCompile(`^c_[0-9a-f]{12}$`).MatchString(b.Columns[0].ID) {
		t.Fatalf("unexpected id format: %s", b.Columns[0].ID)
	}
}

func TestValidate(t *testing.T) {
	cases := map[string]func(b *Board){
		"wrong version":    func(b *Board) { b.Version = 2 },
		"no columns":       func(b *Board) { b.Columns = nil; b.Projects = nil },
		"duplicate id":     func(b *Board) { b.Projects[0].Tasks[0].ID = b.Projects[0].ID },
		"orphan column id": func(b *Board) { b.Projects[0].Tasks[0].ColumnID = "c_missing" },
		"empty column":     func(b *Board) { b.Columns[1].Name = "  " },
		"empty project":    func(b *Board) { b.Projects[0].Name = "" },
		"empty title":      func(b *Board) { b.Projects[0].Tasks[0].Title = " " },
		"missing id":       func(b *Board) { b.Columns[0].ID = "" },
	}
	if err := validBoard().Validate(); err != nil {
		t.Fatalf("valid board rejected: %v", err)
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b := validBoard()
			mutate(&b)
			if err := b.Validate(); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}
