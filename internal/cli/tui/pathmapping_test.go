package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestPathMappingEnterReturnsEditedMapping(t *testing.T) {
	editor := textinput.New()
	model := pathMappingModel{
		rows: []pathRow{
			{Source: "/Users/roy", Status: "[自动]", Target: "/Users/alice"},
			{Source: "/Users/roy/projects/foo", Status: "⚠ 未找到", Target: ""},
		},
		input:      editor,
		sourceHome: "/Users/roy",
		targetHome: "/Users/alice",
	}
	var cmd tea.Cmd
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(pathMappingModel)
	updated, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	model = updated.(pathMappingModel)
	if cmd == nil || !model.editing {
		t.Fatalf("expected editing mode")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/Users/alice/work/foo")})
	model = updated.(pathMappingModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(pathMappingModel)

	result := model.mapping(nil)
	if len(result.ProjectMappings) != 1 || result.ProjectMappings[0].To == "" {
		t.Fatalf("mapping = %#v", result.ProjectMappings)
	}
}

func TestPathMappingViewShowsMissingTargetWarning(t *testing.T) {
	model := pathMappingModel{
		rows: []pathRow{
			{Source: "/Users/roy", Status: "[自动]", Target: "/Users/alice"},
			{Source: "/Users/roy/projects/foo", Status: "⚠ 未找到", Target: ""},
		},
		sourceHome: "/Users/roy",
		targetHome: "/Users/alice",
	}

	view := model.View()
	if !strings.Contains(view, "⚠ 未找到") {
		t.Fatalf("view = %q", view)
	}

	result := model.mapping([]string{"/opt/homebrew/bin/rg"})
	if len(result.ProjectMappings) != 0 {
		t.Fatalf("project mappings = %#v", result.ProjectMappings)
	}
	if result.TargetHome != "/Users/alice" {
		t.Fatalf("target home = %q", result.TargetHome)
	}
	if got, want := result.ExternalPaths, []string{"/opt/homebrew/bin/rg"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("external paths = %#v", got)
	}
}

func TestPathMappingEnterBlockedWhenMissingTargetExists(t *testing.T) {
	model := pathMappingModel{
		rows: []pathRow{
			{Source: "/Users/roy", Status: "[自动]", Target: "/Users/alice"},
			{Source: "/Users/roy/projects/foo", Status: "⚠ 未找到", Target: ""},
		},
		sourceHome: "/Users/roy",
		targetHome: "/Users/alice",
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(pathMappingModel)

	if cmd != nil {
		t.Fatalf("enter should stay in TUI when mapping is incomplete")
	}
	if model.message == "" {
		t.Fatalf("expected validation message")
	}
	if !strings.Contains(model.View(), "请先填写标记为 ⚠ 未找到 的目标路径") {
		t.Fatalf("view = %q", model.View())
	}

	result := model.mapping(nil)
	if len(result.ProjectMappings) != 0 {
		t.Fatalf("project mappings = %#v", result.ProjectMappings)
	}
}
