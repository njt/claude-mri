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
	Watcher   *data.Watcher

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

// fileEventMsg wraps a file event
type fileEventMsg data.FileEvent

// NewModel creates a new model
func NewModel() Model {
	home, _ := os.UserHomeDir()
	basePath := filepath.Join(home, ".claude", "projects")

	m := Model{
		BasePath:   basePath,
		FollowMode: true,
		TreeWidth:  35,
	}

	// Create watcher
	w, err := data.NewWatcher(basePath)
	if err == nil {
		m.Watcher = w
	}

	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.loadProjects,
		tickCmd(),
	}
	if m.Watcher != nil {
		m.Watcher.Start()
		cmds = append(cmds, m.watchFiles)
	}
	return tea.Batch(cmds...)
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

// watchFiles returns file events from the watcher
func (m Model) watchFiles() tea.Msg {
	if m.Watcher == nil {
		return nil
	}
	select {
	case event := <-m.Watcher.Events:
		return fileEventMsg(event)
	case err := <-m.Watcher.Errors:
		return errMsg{err}
	}
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
