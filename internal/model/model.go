package model

import (
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/natdempk/claude-mri/internal/data"
)

// Pane represents which pane has focus
type Pane int

const (
	TreePane Pane = iota
	DetailPane
)

// SortMode represents how the tree is sorted
type SortMode int

const (
	SortAlphabetical SortMode = iota
	SortRecent
)

// Model is the main Bubble Tea model
type Model struct {
	// Data
	Projects  []*data.Project
	Tree      []*TreeNode
	FlatNodes []*TreeNode // flattened visible nodes
	Watcher   *data.Watcher

	// Navigation - Tree pane
	Cursor     int
	Selected   *TreeNode
	TreeScroll int // scroll offset for tree pane

	// Navigation - Detail pane
	Focus              Pane            // which pane has focus
	DetailScroll       int             // scroll offset (in lines) for detail view
	DetailContentHeight int            // total height of detail content (set by view)
	BlockExpanded      map[string]bool // which blocks are expanded (by message UUID)
	DetailExpandAll    bool            // auto-expand new messages when true

	// UI state
	FollowMode bool
	SortMode   SortMode
	Ready      bool
	Width      int
	Height     int
	TreeWidth  int
	DetailView viewport.Model

	// Paths
	BasePath string
}

// TreeHeight returns the visible height of the tree pane
func (m Model) TreeHeight() int {
	return m.Height - 4 // account for header, border, help
}

// fileEventMsg wraps a file event
type fileEventMsg data.FileEvent

// NewModel creates a new model
func NewModel() Model {
	home, _ := os.UserHomeDir()
	basePath := filepath.Join(home, ".claude", "projects")

	m := Model{
		BasePath:      basePath,
		FollowMode:    true,
		TreeWidth:     40,
		Focus:         TreePane,
		BlockExpanded: make(map[string]bool),
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
	}
	if m.Watcher != nil {
		m.Watcher.Start()
		cmds = append(cmds, m.watchFiles)
	}
	return tea.Batch(cmds...)
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
