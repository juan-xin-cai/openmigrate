package tui

import (
	"fmt"
	"io"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type progressMessage struct {
	stage string
	done  bool
	err   error
}

type progressModel struct {
	spinner spinner.Model
	stage   string
	updates <-chan progressMessage
	err     error
	done    bool
}

func RunWithProgress(input io.Reader, output io.Writer, errOut io.Writer, interactive bool, initial string, verbose bool, fn func(update func(string)) error) error {
	if !interactive {
		return fn(func(stage string) {
			fmt.Fprintln(errOut, stage)
		})
	}
	updates := make(chan progressMessage, 8)
	go func() {
		err := fn(func(stage string) {
			if verbose {
				fmt.Fprintf(errOut, "[%s] %s\n", time.Now().Format(time.RFC3339), stage)
			}
			updates <- progressMessage{stage: stage}
		})
		updates <- progressMessage{done: true, err: err}
		close(updates)
	}()
	model := progressModel{
		spinner: spinner.New(),
		stage:   initial,
		updates: updates,
	}
	model.spinner.Spinner = spinner.Line
	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(errOut))
	finalModel, err := program.StartReturningModel()
	if err != nil {
		return err
	}
	return finalModel.(progressModel).err
}

func (m progressModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitForProgress(m.updates))
}

func (m progressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(typed)
		return m, cmd
	case progressMessage:
		if typed.stage != "" {
			m.stage = typed.stage
		}
		if typed.done {
			m.done = true
			m.err = typed.err
			return m, tea.Quit
		}
		return m, waitForProgress(m.updates)
	}
	return m, nil
}

func (m progressModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("%s %s", m.spinner.View(), m.stage)
}

func waitForProgress(ch <-chan progressMessage) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return progressMessage{done: true}
		}
		return msg
	}
}
