package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/natdempk/claude-mri/internal/debug"
	"github.com/natdempk/claude-mri/internal/model"
	"github.com/natdempk/claude-mri/internal/ui"
)

type mainModel struct {
	model.Model
}

func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updatedModel, cmd := m.Model.Update(msg)
	m.Model = updatedModel.(model.Model)
	return m, cmd
}

func (m mainModel) View() string {
	return ui.View(m.Model)
}

func main() {
	// Initialize debug logging
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, "claude-mri-debug.log")
	if err := debug.Init(logPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not init debug log: %v\n", err)
	}
	defer debug.Close()

	m := mainModel{Model: model.NewModel()}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
