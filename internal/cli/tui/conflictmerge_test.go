package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

func TestConflictMergeYAndBApplyExpectedDecisions(t *testing.T) {
	model := conflictModel{rows: []conflictRow{
		{Category: "skills", Key: ".claude/skills/a"},
		{Category: "skills", Key: ".claude/skills/b"},
		{Category: "projects", Key: ".claude/projects/c"},
	}}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	model = updated.(conflictModel)
	if model.rows[0].Action != types.DecisionOverwrite {
		t.Fatalf("row0 action = %v", model.rows[0].Action)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(conflictModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("B")})
	model = updated.(conflictModel)
	if model.rows[0].Action != types.DecisionKeepTarget || model.rows[1].Action != types.DecisionKeepTarget {
		t.Fatalf("rows = %#v", model.rows)
	}
	if model.rows[2].Action != "" {
		t.Fatalf("projects row should stay undecided: %#v", model.rows[2])
	}
}

func TestConflictMergeDoesNotOfferMergeShortcut(t *testing.T) {
	model := conflictModel{rows: []conflictRow{
		{Category: "projects", Key: ".claude/projects/c"},
	}}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("m")})
	model = updated.(conflictModel)
	if model.rows[0].Action != "" {
		t.Fatalf("row0 action = %v", model.rows[0].Action)
	}
	if strings.Contains(model.View(), "[m]") {
		t.Fatalf("view should not mention merge shortcut: %q", model.View())
	}
}
