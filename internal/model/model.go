package model

import (
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/natdempk/claude-mri/internal/data"
)

// Model is the main Bubble Tea model
type Model struct {
	// Data
	Projects  []*data.Project
	Tree      []*TreeNode
	FlatNodes []*TreeNode // flattened visible nodes

	// Navigation
	Cursor   int
	Selected *TreeNode

	// UI state
	FollowMode bool
	Ready      bool
	Width      int
	Height     int
	TreeWidth  int
	DetailView viewport.Model

	// Paths
	BasePath string
}

// tickMsg triggers periodic updates
type tickMsg time.Time

// NewModel creates a new model
func NewModel() Model {
	home, _ := os.UserHomeDir()
	basePath := filepath.Join(home, ".claude", "projects")

	return Model{
		BasePath:   basePath,
		FollowMode: true,
		TreeWidth:  35,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadProjects,
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) loadProjects() tea.Msg {
	projects, err := data.ScanProjects(m.BasePath)
	if err != nil {
		return errMsg{err}
	}
	return projectsLoadedMsg{projects}
}

type projectsLoadedMsg struct {
	projects []*data.Project
}

type errMsg struct {
	err error
}

// View renders the model (stub - actual rendering in ui package)
func (m Model) View() string {
	return "Loading..."
}
