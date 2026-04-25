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

func TestPathMappingViewShowsSkipMarker(t *testing.T) {
	model := pathMappingModel{
		rows: []pathRow{
			{Source: "/Users/roy", Status: "[自动]", Target: "/Users/alice"},
			{Source: "/Users/roy/projects/foo", Status: statusFor("", false), Target: ""},
		},
		sourceHome: "/Users/roy",
		targetHome: "/Users/alice",
	}

	view := model.View()
	if !strings.Contains(view, "⤼ 跳过") {
		t.Fatalf("view = %q", view)
	}
	if !strings.Contains(view, "home 兜底") {
		t.Fatalf("view should explain home fallback, got %q", view)
	}

	result := model.mapping([]string{"/opt/homebrew/bin/rg"})
	if len(result.ProjectMappings) != 0 {
		t.Fatalf("empty target rows must be dropped, got %#v", result.ProjectMappings)
	}
	if result.TargetHome != "/Users/alice" {
		t.Fatalf("target home = %q", result.TargetHome)
	}
	if got, want := result.ExternalPaths, []string{"/opt/homebrew/bin/rg"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("external paths = %#v", got)
	}
}

func TestPathMappingEnterPassesThroughWithEmptyTargets(t *testing.T) {
	model := pathMappingModel{
		rows: []pathRow{
			{Source: "/Users/roy", Status: "[自动]", Target: "/Users/alice"},
			{Source: "/Users/roy/projects/foo", Status: statusFor("", false), Target: ""},
			{Source: "/Users/roy/projects/bar", Status: statusFor("", false), Target: ""},
		},
		sourceHome: "/Users/roy",
		targetHome: "/Users/alice",
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(pathMappingModel)

	if cmd == nil {
		t.Fatalf("enter must accept the mapping even with empty targets")
	}
	if model.message != "" {
		t.Fatalf("no validation message expected, got %q", model.message)
	}

	result := model.mapping(nil)
	if len(result.ProjectMappings) != 0 {
		t.Fatalf("skipped rows should not contribute mappings, got %#v", result.ProjectMappings)
	}
	if result.TargetHome != "/Users/alice" {
		t.Fatalf("target home = %q", result.TargetHome)
	}
}

func TestPathMappingEditingTogglesSkipStatus(t *testing.T) {
	editor := textinput.New()
	model := pathMappingModel{
		rows: []pathRow{
			{Source: "/Users/roy", Status: "[自动]", Target: "/Users/alice"},
			{Source: "/Users/roy/projects/foo", Status: "[已匹配]", Target: "/Users/alice/projects/foo"},
		},
		input:      editor,
		sourceHome: "/Users/roy",
		targetHome: "/Users/alice",
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(pathMappingModel)
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	model = updated.(pathMappingModel)
	model.input.SetValue("")
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(pathMappingModel)

	if !strings.Contains(model.rows[1].Status, "⤼ 跳过") {
		t.Fatalf("clearing target should mark skip, status = %q", model.rows[1].Status)
	}
}
