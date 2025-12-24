package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/natdempk/claude-mri/internal/data"
	"github.com/natdempk/claude-mri/internal/model"
)

// View renders the UI
func View(m model.Model) string {
	if !m.Ready {
		return "Loading..."
	}

	// Header
	followStatus := "[F]ollow: OFF"
	if m.FollowMode {
		followStatus = FollowOnStyle.Render("[F]ollow: ON")
	} else {
		followStatus = FollowOffStyle.Render("[F]ollow: OFF")
	}
	header := HeaderStyle.Render("claude-mri") +
		strings.Repeat(" ", m.Width-20-len("claude-mri")) +
		followStatus

	// Tree pane
	treeContent := renderTree(m)
	treePane := TreePaneStyle.
		Height(m.Height - 4).
		Render(treeContent)

	// Detail pane
	detailContent := renderDetails(m)
	detailPane := DetailPaneStyle.
		Width(m.Width - m.TreeWidth - 4).
		Height(m.Height - 4).
		Render(detailContent)

	// Combine panes
	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, detailPane)

	// Help bar
	help := HelpStyle.Render("j/k:navigate  Enter:expand  f:follow  q:quit")

	return lipgloss.JoinVertical(lipgloss.Left, header, body, help)
}

func renderTree(m model.Model) string {
	var sb strings.Builder

	visibleHeight := m.TreeHeight()
	startIdx := m.TreeScroll
	endIdx := startIdx + visibleHeight
	if endIdx > len(m.FlatNodes) {
		endIdx = len(m.FlatNodes)
	}

	for i := startIdx; i < endIdx; i++ {
		node := m.FlatNodes[i]

		// Indentation
		indent := strings.Repeat("  ", node.Depth())

		// Expand indicator
		indicator := "  "
		if len(node.Children) > 0 {
			if node.Expanded {
				indicator = "▼ "
			} else {
				indicator = "▶ "
			}
		}

		// Label
		label := indent + indicator + node.Label

		// Style based on selection
		if i == m.Cursor {
			label = SelectedStyle.Render(label)
		} else {
			label = TreeItemStyle.Render(label)
		}

		sb.WriteString(label)
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderDetails(m model.Model) string {
	if m.Selected == nil {
		return "Select an item to view details"
	}

	var sb strings.Builder

	switch m.Selected.Type {
	case model.NodeProject:
		sb.WriteString(fmt.Sprintf("Project: %s\n", m.Selected.Label))
		sb.WriteString(fmt.Sprintf("Path: %s\n", m.Selected.ID))
		sb.WriteString(fmt.Sprintf("Sessions: %d\n", len(m.Selected.Children)))

	case model.NodeSession:
		sb.WriteString(fmt.Sprintf("Session: %s\n", m.Selected.ID))
		if m.Selected.Session != nil {
			sb.WriteString(fmt.Sprintf("Messages: %d\n", len(m.Selected.Session.Messages)))
			sb.WriteString(fmt.Sprintf("Updated: %s\n", m.Selected.Session.UpdatedAt.Format("2006-01-02 15:04:05")))
		}

	case model.NodeMessage:
		if m.Selected.Message != nil {
			msg := m.Selected.Message
			sb.WriteString(fmt.Sprintf("Type: %s\n", msg.Type))
			sb.WriteString(fmt.Sprintf("Time: %s\n\n", msg.Timestamp.Format("15:04:05")))

			for _, block := range msg.Blocks {
				sb.WriteString(renderBlock(&block))
				sb.WriteString("\n")
			}
		}

	case model.NodeBlock:
		if m.Selected.Block != nil {
			sb.WriteString(renderBlock(m.Selected.Block))
		}
	}

	return sb.String()
}

func renderBlock(b *data.ContentBlock) string {
	var sb strings.Builder

	switch b.Type {
	case "thinking":
		sb.WriteString(ThinkingStyle.Render("[thinking]\n"))
		lines := strings.Split(b.Thinking, "\n")
		if len(lines) > 5 {
			sb.WriteString(ThinkingStyle.Render(strings.Join(lines[:5], "\n")))
			sb.WriteString(ThinkingStyle.Render(fmt.Sprintf("\n[+%d more lines]", len(lines)-5)))
		} else {
			sb.WriteString(ThinkingStyle.Render(b.Thinking))
		}

	case "text":
		sb.WriteString("[text]\n")
		sb.WriteString(b.Text)

	case "tool_use":
		sb.WriteString(ToolNameStyle.Render("[tool_use] " + b.ToolName))
		sb.WriteString("\n")
		if b.ToolInput != "" {
			sb.WriteString(b.ToolInput)
		}

	case "tool_result":
		sb.WriteString("[tool_result]\n")
		result := b.Result
		if len(result) > 500 {
			result = result[:500] + "\n[truncated...]"
		}
		sb.WriteString(result)
	}

	return sb.String()
}
