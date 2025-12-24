package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/natdempk/claude-mri/internal/model"
	"github.com/natdempk/claude-mri/internal/ui"
)

type mainModel struct {
	model.Model
}

func (m mainModel) View() string {
	return ui.View(m.Model)
}

func main() {
	m := mainModel{Model: model.NewModel()}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
