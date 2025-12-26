package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/natdempk/claude-mri/internal/data"
	"github.com/natdempk/claude-mri/internal/debug"
)

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer debug.Time(fmt.Sprintf("Update(%T)", msg))()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		debug.Log("KeyMsg: %s", msg.String())
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.DetailView = viewport.New(msg.Width-m.TreeWidth-4, msg.Height-4)
		m.Ready = true
		m.UpdateDetailContentHeight() // Recalc since width affects wrapping
		return m, nil

	case projectsLoadedMsg:
		debug.Log("projectsLoadedMsg: %d projects", len(msg.projects))
		defer debug.Time("projectsLoadedMsg processing")()
		// Preserve expanded state before rebuild
		expanded := m.getExpandedIDs()
		selectedID := ""
		if m.Selected != nil {
			selectedID = m.Selected.ID
		}

		m.Projects = msg.projects
		m.sortProjects() // Apply current sort mode
		m.Tree = BuildTree(m.Projects)

		// Restore expanded state
		m.restoreExpandedState(expanded)
		// Reload messages for expanded sessions (they have new session objects)
		m.reloadExpandedSessions()
		m.flattenTree()

		// Restore selection
		if selectedID != "" {
			for i, node := range m.FlatNodes {
				if node.ID == selectedID {
					m.Cursor = i
					m.Selected = node
					break
				}
			}
		} else if len(m.FlatNodes) > 0 {
			m.Selected = m.FlatNodes[0]
		}
		// Reload messages for selected session (it's a new object after rebuild)
		m.loadSelectedSession()
		m.ensureCursorVisible()

		// In follow mode, scroll detail pane to end (but keep tree selection stable)
		if m.FollowMode {
			m.scrollDetailToEnd()
		}
		return m, nil

	case fileEventMsg:
		debug.Log("fileEventMsg: %s", msg.Path)
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

	case "tab":
		// Switch focus between panes
		if m.Focus == TreePane {
			// Only switch to detail if we have a session selected
			if m.Selected != nil && m.Selected.Type == NodeSession {
				m.Focus = DetailPane
				m.DetailScroll = 0
			}
		} else {
			m.Focus = TreePane
		}

	case "f":
		m.FollowMode = !m.FollowMode
		if m.FollowMode {
			m.scrollToEnd()
		}

	case "s":
		// Toggle sort mode
		if m.SortMode == SortAlphabetical {
			m.SortMode = SortRecent
		} else {
			m.SortMode = SortAlphabetical
		}
		m.sortAndRebuildTree()

	default:
		// Handle pane-specific keys
		if m.Focus == TreePane {
			return m.handleTreeKey(msg)
		} else {
			return m.handleDetailKey(msg)
		}
	}

	return m, nil
}

func (m Model) handleTreeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.Cursor < len(m.FlatNodes)-1 {
			m.Cursor++
			m.Selected = m.FlatNodes[m.Cursor]
			m.ensureCursorVisible()
			m.loadSelectedSession()
			// Reset detail view when selection changes
			m.DetailScroll = 0
			m.DetailExpandAll = false
		}

	case "k", "up":
		if m.Cursor > 0 {
			m.Cursor--
			m.Selected = m.FlatNodes[m.Cursor]
			m.ensureCursorVisible()
			m.loadSelectedSession()
			// Reset detail view when selection changes
			m.DetailScroll = 0
			m.DetailExpandAll = false
		}

	case "enter", "l", "right":
		if m.Selected != nil {
			if m.Selected.Type == NodeSession {
				// For sessions: load messages and switch to detail pane
				m.loadSelectedSession()
				m.Focus = DetailPane
				m.DetailScroll = 0
			} else if m.Selected.IsExpandable() {
				// For other expandable nodes: toggle expansion
				m.Selected.Expanded = !m.Selected.Expanded
				m.flattenTree()
				m.ensureCursorVisible()
			}
		}

	case "h", "left", "esc":
		if m.Selected != nil && m.Selected.Expanded {
			m.Selected.Expanded = false
			m.flattenTree()
			m.ensureCursorVisible()
		}

	case "home":
		if len(m.FlatNodes) > 0 {
			m.Cursor = 0
			m.Selected = m.FlatNodes[0]
			m.TreeScroll = 0
			m.loadSelectedSession()
			m.DetailScroll = 0
		}

	case "end":
		if len(m.FlatNodes) > 0 {
			m.Cursor = len(m.FlatNodes) - 1
			m.Selected = m.FlatNodes[m.Cursor]
			m.ensureCursorVisible()
			m.loadSelectedSession()
		}
	}

	return m, nil
}

func (m Model) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	messages := m.getSelectedMessages()
	if len(messages) == 0 {
		return m, nil
	}

	switch msg.String() {
	case "j", "down":
		// Scroll down by 1 line - View will clamp if past end
		m.DetailScroll++

	case "k", "up":
		// Scroll up by 1 line
		if m.DetailScroll > 0 {
			m.DetailScroll--
		}

	case "g", "home":
		// Go to top
		m.DetailScroll = 0

	case "G", "end":
		// Go to bottom - use large value, View will clamp to actual content length
		m.DetailScroll = 1000000

	case "ctrl+d", "pgdown":
		// Page down - don't cap here, View will clamp
		m.DetailScroll += m.DetailHeight() / 2

	case "ctrl+u", "pgup":
		// Page up
		m.DetailScroll -= m.DetailHeight() / 2
		if m.DetailScroll < 0 {
			m.DetailScroll = 0
		}

	case "enter", "l", "right":
		// Expand all messages and enable auto-expand for new ones
		m.DetailExpandAll = true
		for _, msg := range messages {
			m.BlockExpanded[msg.UUID] = true
		}
		m.UpdateDetailContentHeight()

	case "h", "left":
		// Collapse all messages and disable auto-expand
		m.DetailExpandAll = false
		for _, msg := range messages {
			m.BlockExpanded[msg.UUID] = false
		}
		m.UpdateDetailContentHeight()

	case "esc":
		// Return to tree pane
		m.Focus = TreePane
	}

	return m, nil
}

// getSelectedMessages returns messages for the currently selected session
func (m *Model) getSelectedMessages() []*data.Message {
	if m.Selected == nil || m.Selected.Type != NodeSession {
		return nil
	}
	if m.Selected.Session == nil {
		return nil
	}
	return m.Selected.Session.Messages
}

// DetailHeight returns visible height of detail pane
func (m Model) DetailHeight() int {
	return m.Height - 4
}

// scrollToEnd scrolls both panes to the end
func (m *Model) scrollToEnd() {
	// Scroll tree to end
	if len(m.FlatNodes) > 0 {
		m.Cursor = len(m.FlatNodes) - 1
		m.Selected = m.FlatNodes[m.Cursor]
		m.ensureCursorVisible()
		m.loadSelectedSession()
	}

	// Scroll detail to end - use large value, View will clamp
	m.DetailScroll = 1000000
}

// scrollDetailToEnd scrolls only the detail pane to the end (for follow mode)
func (m *Model) scrollDetailToEnd() {
	// Use large value - View will clamp to actual content length
	m.DetailScroll = 1000000
}

// scrollToStart scrolls both panes to the start
func (m *Model) scrollToStart() {
	// Scroll tree to start
	if len(m.FlatNodes) > 0 {
		m.Cursor = 0
		m.Selected = m.FlatNodes[0]
		m.TreeScroll = 0
		m.loadSelectedSession()
	}

	// Scroll detail to start
	m.DetailScroll = 0
}

// sortAndRebuildTree sorts projects based on current SortMode and rebuilds tree
func (m *Model) sortAndRebuildTree() {
	m.sortProjects()

	// Preserve state
	expanded := m.getExpandedIDs()
	selectedID := ""
	if m.Selected != nil {
		selectedID = m.Selected.ID
	}

	// Rebuild tree
	m.Tree = BuildTree(m.Projects)
	m.restoreExpandedState(expanded)
	m.reloadExpandedSessions()
	m.flattenTree()

	// Restore selection
	if selectedID != "" {
		for i, node := range m.FlatNodes {
			if node.ID == selectedID {
				m.Cursor = i
				m.Selected = node
				break
			}
		}
	}
	m.loadSelectedSession()
	m.ensureCursorVisible()
}

// sortProjects sorts the projects list based on current SortMode
func (m *Model) sortProjects() {
	switch m.SortMode {
	case SortRecent:
		// Sort by most recent session update time
		sort.Slice(m.Projects, func(i, j int) bool {
			iTime := m.Projects[i].MostRecentUpdate()
			jTime := m.Projects[j].MostRecentUpdate()
			return iTime.After(jTime)
		})
	case SortAlphabetical:
		sort.Slice(m.Projects, func(i, j int) bool {
			return m.Projects[i].Name < m.Projects[j].Name
		})
	}
}

// UpdateDetailContentHeight recalculates the total content height
func (m *Model) UpdateDetailContentHeight() {
	messages := m.getSelectedMessages()
	if len(messages) == 0 {
		m.DetailContentHeight = 0
		return
	}

	// Calculate available width for content (matches view.go calculation)
	maxWidth := m.Width - m.TreeWidth - 8
	if maxWidth < 20 {
		maxWidth = 20
	}

	// Count lines including wrapped lines
	totalLines := 0
	for _, msg := range messages {
		// Header line
		totalLines++

		if m.BlockExpanded[msg.UUID] {
			// Expanded: count actual lines in blocks with wrapping
			for _, block := range msg.Blocks {
				var content string
				switch block.Type {
				case "text":
					content = block.Text
					totalLines++ // label line
				case "thinking":
					content = block.Thinking
					totalLines++ // label line
				case "tool_use":
					content = block.ToolInput
					totalLines += 2 // tool name + label
				case "tool_result":
					content = block.Result
					totalLines += 2 // label lines
				}
				// Count lines with wrapping estimation
				totalLines += countWrappedLines(content, maxWidth-6) // indent
			}
		} else {
			// Collapsed: just preview line
			totalLines++
		}
		totalLines++ // blank line between messages
	}
	m.DetailContentHeight = totalLines
}

// countWrappedLines estimates how many display lines a string will take
func countWrappedLines(s string, width int) int {
	if width <= 0 {
		return 1
	}
	lines := 0
	for _, line := range strings.Split(s, "\n") {
		if len(line) == 0 {
			lines++
			continue
		}
		// Estimate wrapped lines (rough: assumes 1 byte = 1 char width)
		wrappedCount := (len(line) + width - 1) / width
		if wrappedCount < 1 {
			wrappedCount = 1
		}
		lines += wrappedCount
	}
	return lines
}

// loadSelectedSession loads messages for the selected session if needed
func (m *Model) loadSelectedSession() {
	if m.Selected == nil {
		return
	}
	if m.Selected.Type == NodeSession && m.Selected.Session != nil {
		if len(m.Selected.Session.Messages) == 0 {
			data.LoadSession(m.Selected.Session)
		}
		// Auto-expand new messages if in expand-all mode
		if m.DetailExpandAll {
			for _, msg := range m.Selected.Session.Messages {
				m.BlockExpanded[msg.UUID] = true
			}
		}
		m.UpdateDetailContentHeight()
	}
}

// reloadExpandedSessions reloads messages for all expanded session nodes
// This is needed after tree rebuild since session objects are recreated
func (m *Model) reloadExpandedSessions() {
	for _, node := range m.Tree {
		m.reloadExpandedSessionsRecursive(node)
	}
}

func (m *Model) reloadExpandedSessionsRecursive(node *TreeNode) {
	if node.Type == NodeSession && node.Expanded && node.Session != nil {
		// Reload messages if not loaded
		if len(node.Session.Messages) == 0 {
			data.LoadSession(node.Session)
		}
		// Rebuild children from messages
		if len(node.Children) == 0 && len(node.Session.Messages) > 0 {
			for _, msg := range node.Session.Messages {
				node.Children = append(node.Children, BuildMessageNode(msg))
			}
		}
	}
	// Recurse into children (for expanded projects)
	for _, child := range node.Children {
		m.reloadExpandedSessionsRecursive(child)
	}
}

// rebuildSessionChildren rebuilds children nodes after loading messages
func (m *Model) rebuildSessionChildren() {
	if m.Selected == nil {
		return
	}
	if m.Selected.Type == NodeSession && m.Selected.Session != nil {
		if len(m.Selected.Children) == 0 && len(m.Selected.Session.Messages) > 0 {
			for _, msg := range m.Selected.Session.Messages {
				m.Selected.Children = append(m.Selected.Children, BuildMessageNode(msg))
			}
		}
	}
}

// ensureCursorVisible adjusts TreeScroll so cursor is visible
func (m *Model) ensureCursorVisible() {
	visibleHeight := m.TreeHeight()
	if visibleHeight <= 0 {
		visibleHeight = 20 // fallback before window size known
	}

	// Scroll down if cursor below visible area
	if m.Cursor >= m.TreeScroll+visibleHeight {
		m.TreeScroll = m.Cursor - visibleHeight + 1
	}
	// Scroll up if cursor above visible area
	if m.Cursor < m.TreeScroll {
		m.TreeScroll = m.Cursor
	}
	// Don't scroll past the end
	maxScroll := len(m.FlatNodes) - visibleHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.TreeScroll > maxScroll {
		m.TreeScroll = maxScroll
	}
	if m.TreeScroll < 0 {
		m.TreeScroll = 0
	}
}

// getExpandedIDs collects IDs of all expanded nodes
func (m *Model) getExpandedIDs() map[string]bool {
	expanded := make(map[string]bool)
	for _, node := range m.Tree {
		m.collectExpanded(node, expanded)
	}
	return expanded
}

func (m *Model) collectExpanded(node *TreeNode, expanded map[string]bool) {
	if node.Expanded {
		expanded[node.ID] = true
	}
	for _, child := range node.Children {
		m.collectExpanded(child, expanded)
	}
}

// restoreExpandedState marks nodes as expanded if they were before
func (m *Model) restoreExpandedState(expanded map[string]bool) {
	for _, node := range m.Tree {
		m.restoreExpanded(node, expanded)
	}
}

func (m *Model) restoreExpanded(node *TreeNode, expanded map[string]bool) {
	if expanded[node.ID] {
		node.Expanded = true
	}
	for _, child := range node.Children {
		m.restoreExpanded(child, expanded)
	}
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
