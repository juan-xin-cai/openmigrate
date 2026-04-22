package tui

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/openmigrate/openmigrate/internal/core/types"
)

type conflictRow struct {
	Category string
	Key      string
	Action   types.DecisionAction
}

type conflictModel struct {
	rows     []conflictRow
	cursor   int
	canceled bool
}

func RunConflictMerge(input io.Reader, output io.Writer, report types.ConflictReport, defaults types.ConflictDecision) (types.ConflictDecision, error) {
	rows := flattenConflicts(report, defaults)
	program := tea.NewProgram(conflictModel{rows: rows}, tea.WithInput(input), tea.WithOutput(output))
	finalModel, err := program.StartReturningModel()
	if err != nil {
		return types.ConflictDecision{}, err
	}
	model := finalModel.(conflictModel)
	if model.canceled {
		return types.ConflictDecision{}, errUserCanceled
	}
	actions := make(map[string]types.DecisionAction, len(model.rows))
	for _, row := range model.rows {
		if row.Action != "" {
			actions[row.Key] = row.Action
		}
	}
	return types.ConflictDecision{Actions: actions}, nil
}

func (m conflictModel) Init() tea.Cmd { return nil }

func (m conflictModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case "y":
			m.rows[m.cursor].Action = types.DecisionOverwrite
		case "n":
			m.rows[m.cursor].Action = types.DecisionKeepTarget
		case "s":
			m.rows[m.cursor].Action = types.DecisionSkip
		case "A":
			category := m.rows[m.cursor].Category
			for i := range m.rows {
				if m.rows[i].Category == category {
					m.rows[i].Action = types.DecisionOverwrite
				}
			}
		case "B":
			category := m.rows[m.cursor].Category
			for i := range m.rows {
				if m.rows[i].Category == category {
					m.rows[i].Action = types.DecisionKeepTarget
				}
			}
		case "enter":
			if m.allDecided() {
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

func (m conflictModel) View() string {
	if len(m.rows) == 0 {
		return "无冲突\n"
	}
	var b strings.Builder
	b.WriteString("冲突处理\n\n")
	for i, row := range m.rows {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		action := actionLabel(row.Action)
		if row.Action == "" {
			action = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("未决策")
		}
		fmt.Fprintf(&b, "%s[%s] %s => %s\n", cursor, row.Category, row.Key, action)
	}
	b.WriteString("\n[y] 用源  [n] 保留目标  [s] 跳过  [A/B] 当前类别批量  [回车] 完成  [q] 取消\n")
	return b.String()
}

func (m conflictModel) allDecided() bool {
	for _, row := range m.rows {
		if row.Action == "" {
			return false
		}
	}
	return true
}

func flattenConflicts(report types.ConflictReport, defaults types.ConflictDecision) []conflictRow {
	rows := make([]conflictRow, 0, 16)
	for category, bucket := range report.Buckets {
		for _, item := range bucket.Conflicts {
			rows = append(rows, conflictRow{
				Category: category,
				Key:      item.Key,
				Action:   defaults.Actions[item.Key],
			})
		}
	}
	return rows
}

func actionLabel(action types.DecisionAction) string {
	switch action {
	case types.DecisionOverwrite:
		return "使用源"
	case types.DecisionKeepTarget:
		return "保留目标"
	case types.DecisionSkip:
		return "跳过"
	default:
		return ""
	}
}
