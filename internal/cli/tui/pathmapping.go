package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

type pathRow struct {
	Source string
	Status string
	Target string
}

type pathMappingModel struct {
	rows       []pathRow
	cursor     int
	input      textinput.Model
	editing    bool
	canceled   bool
	message    string
	sourceHome string
	targetHome string
}

func RunPathMapping(input io.Reader, output io.Writer, preview types.ImportPreview) (types.PathMapping, error) {
	rows := make([]pathRow, 0, len(preview.PathScan.ProjectRoots)+1)
	if preview.PathScan.HomePrefix != "" {
		rows = append(rows, pathRow{
			Source: preview.PathScan.HomePrefix,
			Status: "[自动]",
			Target: preview.SuggestedMapping.TargetHome,
		})
	}
	for _, pair := range preview.SuggestedMapping.ProjectMappings {
		rows = append(rows, pathRow{Source: pair.From, Status: statusFor(pair.To, false), Target: pair.To})
	}
	editor := textinput.New()
	editor.Placeholder = "输入目标路径"
	model := pathMappingModel{
		rows:       rows,
		input:      editor,
		sourceHome: preview.PathScan.HomePrefix,
		targetHome: preview.SuggestedMapping.TargetHome,
	}
	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(output))
	finalModel, err := program.StartReturningModel()
	if err != nil {
		return types.PathMapping{}, err
	}
	result := finalModel.(pathMappingModel)
	if result.canceled {
		return types.PathMapping{}, errUserCanceled
	}
	return result.mapping(preview.PathScan.ExternalPaths), nil
}

var errUserCanceled = fmt.Errorf("user canceled")

func (m pathMappingModel) Init() tea.Cmd { return textinput.Blink }

func (m pathMappingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editing {
		switch typed := msg.(type) {
		case tea.KeyMsg:
			switch typed.String() {
			case "enter":
				m.rows[m.cursor].Target = strings.TrimSpace(m.input.Value())
				m.rows[m.cursor].Status = statusFor(m.rows[m.cursor].Target, m.cursor == 0)
				m.message = ""
				m.editing = false
				return m, nil
			case "esc":
				m.message = ""
				m.editing = false
				return m, nil
			}
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "q":
			m.canceled = true
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case " ":
			m.input.SetValue(m.rows[m.cursor].Target)
			m.input.Focus()
			m.editing = true
			return m, textinput.Blink
		case "a", "A":
			m.rows = append(m.rows, pathRow{Status: statusFor("", false)})
			m.cursor = len(m.rows) - 1
			m.input.SetValue("")
			m.message = ""
			m.editing = true
			return m, textinput.Blink
		case "enter":
			m.message = ""
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pathMappingModel) View() string {
	if len(m.rows) == 0 {
		return "无路径映射需要确认\n"
	}
	var b strings.Builder
	b.WriteString("路径映射\n\n")
	for i, row := range m.rows {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		status := row.Status
		if strings.Contains(status, "⤼") {
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(status)
		}
		fmt.Fprintf(&b, "%s%s\n  源: %s\n  目标: %s\n", cursor, status, row.Source, row.Target)
		if i == m.cursor && m.editing {
			fmt.Fprintf(&b, "  编辑: %s\n", m.input.View())
		}
	}
	if m.message != "" {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(m.message))
		b.WriteString("\n")
	}
	b.WriteString("\n留空表示跳过该项（按 home 前缀兜底替换）\n")
	b.WriteString("[↑/↓] 选择  [空格] 编辑  [A] 新增  [回车] 确认  [q] 取消\n")
	return b.String()
}

func statusFor(target string, isHomeRow bool) string {
	if strings.TrimSpace(target) == "" {
		return "⤼ 跳过（home 兜底）"
	}
	if isHomeRow {
		return "[自动]"
	}
	return "[已匹配]"
}

func (m pathMappingModel) mapping(external []string) types.PathMapping {
	result := types.PathMapping{
		SourceHome:    m.sourceHome,
		TargetHome:    m.targetHome,
		ExternalPaths: append([]string(nil), external...),
	}
	for i, row := range m.rows {
		if i == 0 && row.Source == m.sourceHome {
			if row.Target != "" {
				result.TargetHome = row.Target
			}
			continue
		}
		if row.Source == "" || row.Target == "" {
			continue
		}
		result.ProjectMappings = append(result.ProjectMappings, types.PathPair{From: row.Source, To: row.Target})
	}
	return result
}
