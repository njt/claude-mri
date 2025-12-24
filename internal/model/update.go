package model

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/natdempk/claude-mri/internal/data"
)

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.DetailView = viewport.New(msg.Width-m.TreeWidth-4, msg.Height-4)
		m.Ready = true
		return m, nil

	case projectsLoadedMsg:
		m.Projects = msg.projects
		m.Tree = BuildTree(msg.projects)
		m.flattenTree()
		if len(m.FlatNodes) > 0 {
			m.Selected = m.FlatNodes[0]
		}
		return m, nil

	case tickMsg:
		// Refresh data periodically
		return m, tea.Batch(m.loadProjects, tickCmd())

	case fileEventMsg:
		// Reload projects on file change
		// In follow mode, this will auto-update the tree
		return m, tea.Batch(m.loadProjects, m.watchFiles)

	case errMsg:
		// Could display error, for now just ignore
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.Watcher != nil {
			m.Watcher.Stop()
		}
		return m, tea.Quit

	case "j", "down":
		if m.Cursor < len(m.FlatNodes)-1 {
			m.Cursor++
			m.Selected = m.FlatNodes[m.Cursor]
			m.updateDetailView()
		}

	case "k", "up":
		if m.Cursor > 0 {
			m.Cursor--
			m.Selected = m.FlatNodes[m.Cursor]
			m.updateDetailView()
		}

	case "enter", "l", "right":
		if m.Selected != nil && len(m.Selected.Children) > 0 {
			m.Selected.Expanded = !m.Selected.Expanded
			// Load session messages if needed
			if m.Selected.Type == NodeSession && m.Selected.Session != nil {
				if len(m.Selected.Session.Messages) == 0 {
					data.LoadSession(m.Selected.Session)
					// Rebuild this node's children
					m.Selected.Children = nil
					for _, msg := range m.Selected.Session.Messages {
						m.Selected.Children = append(m.Selected.Children, BuildMessageNode(msg))
					}
				}
			}
			m.flattenTree()
		}

	case "h", "left", "esc":
		if m.Selected != nil && m.Selected.Expanded {
			m.Selected.Expanded = false
			m.flattenTree()
		}

	case "f":
		m.FollowMode = !m.FollowMode
	}

	return m, nil
}

// flattenTree creates a flat list of visible nodes
func (m *Model) flattenTree() {
	m.FlatNodes = nil
	for _, node := range m.Tree {
		m.flattenNode(node, 0)
	}
	// Adjust cursor if needed
	if m.Cursor >= len(m.FlatNodes) {
		m.Cursor = len(m.FlatNodes) - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if len(m.FlatNodes) > 0 {
		m.Selected = m.FlatNodes[m.Cursor]
	}
}

func (m *Model) flattenNode(node *TreeNode, depth int) {
	node.depth = depth
	m.FlatNodes = append(m.FlatNodes, node)
	if node.Expanded {
		for _, child := range node.Children {
			m.flattenNode(child, depth+1)
		}
	}
}

func (m *Model) updateDetailView() {
	// Detail view content is rendered on-demand in View
}
