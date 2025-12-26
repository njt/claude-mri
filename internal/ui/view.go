package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/natdempk/claude-mri/internal/data"
	"github.com/natdempk/claude-mri/internal/debug"
	"github.com/natdempk/claude-mri/internal/model"
)

// View renders the UI
func View(m model.Model) string {
	defer debug.Time("View")()

	if !m.Ready {
		return "Loading..."
	}

	// Header
	focusIndicator := "[Tree]"
	if m.Focus == model.DetailPane {
		focusIndicator = "[Chat]"
	}
	sortIndicator := "[A-Z]"
	if m.SortMode == model.SortRecent {
		sortIndicator = "[Recent]"
	}
	followStatus := "[F]ollow: OFF"
	if m.FollowMode {
		followStatus = FollowOnStyle.Render("[F]ollow: ON")
	} else {
		followStatus = FollowOffStyle.Render("[F]ollow: OFF")
	}
	header := HeaderStyle.Render("claude-mri") +
		"  " + focusIndicator + " " + sortIndicator +
		strings.Repeat(" ", max(0, m.Width-35-len("claude-mri")-len(focusIndicator)-len(sortIndicator))) +
		followStatus

	// Tree pane
	treeContent := renderTree(m)
	treePaneStyle := TreePaneStyle
	if m.Focus == model.TreePane {
		treePaneStyle = TreePaneFocusedStyle
	}
	treePane := treePaneStyle.
		Height(m.Height - 4).
		Render(treeContent)

	// Detail pane
	detailContent := renderConversation(m)
	detailPaneStyle := DetailPaneStyle
	if m.Focus == model.DetailPane {
		detailPaneStyle = DetailPaneFocusedStyle
	}
	detailPane := detailPaneStyle.
		Width(m.Width - m.TreeWidth - 4).
		Height(m.Height - 4).
		Render(detailContent)

	// Combine panes
	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, detailPane)

	// Help bar
	help := HelpStyle.Render("Tab:switch  j/k:nav  Enter:expand  s:sort  f:follow  q:quit")

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
		if node.IsExpandable() {
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

func renderConversation(m model.Model) string {
	if m.Selected == nil {
		return "Select a session to view conversation"
	}

	// Show session info for non-session nodes
	if m.Selected.Type != model.NodeSession {
		return renderNodeInfo(m)
	}

	// Get messages
	if m.Selected.Session == nil {
		return "No session data"
	}
	messages := m.Selected.Session.Messages
	if len(messages) == 0 {
		return "No messages in session"
	}

	// Render all messages
	var allLines []string
	maxWidth := m.Width - m.TreeWidth - 8
	if maxWidth < 20 {
		maxWidth = 20
	}
	for _, msg := range messages {
		isExpanded := m.BlockExpanded[msg.UUID]
		msgStr := renderMessage(msg, false, isExpanded, maxWidth)
		// Split and truncate each line to prevent layout breakage
		for _, line := range strings.Split(msgStr, "\n") {
			if lipgloss.Width(line) > maxWidth {
				line = truncateWidth(line, maxWidth)
			}
			allLines = append(allLines, line)
		}
		allLines = append(allLines, "") // blank line between messages
	}

	// Reserve 2 lines for scroll indicators (always present for stable layout)
	visibleHeight := m.DetailHeight() - 2
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	// Apply scroll offset
	startLine := m.DetailScroll
	// Clamp so we show last visibleHeight lines when scrolled past end
	maxStart := len(allLines) - visibleHeight
	if maxStart < 0 {
		maxStart = 0
	}
	if startLine > maxStart {
		startLine = maxStart
	}
	endLine := startLine + visibleHeight
	if endLine > len(allLines) {
		endLine = len(allLines)
	}

	// Build output with fixed indicator lines
	var result strings.Builder

	// Top indicator (always present)
	if startLine > 0 {
		result.WriteString(fmt.Sprintf("  ↑ %d lines above", startLine))
	}
	result.WriteString("\n")

	// Content
	visibleLines := allLines[startLine:endLine]
	result.WriteString(strings.Join(visibleLines, "\n"))

	// Bottom indicator (always on its own line)
	remaining := len(allLines) - endLine
	result.WriteString("\n")
	if remaining > 0 {
		result.WriteString(fmt.Sprintf("  ↓ %d lines below", remaining))
	}

	return result.String()
}

func renderMessage(msg *data.Message, isSelected, isExpanded bool, maxWidth int) string {
	var sb strings.Builder

	// Message header with icon and type
	icon := "👤"
	headerStyle := UserMessageStyle
	if msg.Type == "assistant" {
		icon = "🤖"
		headerStyle = AssistantMessageStyle
	}

	header := fmt.Sprintf("%s %s", icon, msg.Type)
	if msg.Timestamp.Year() > 1 {
		header += fmt.Sprintf(" (%s)", msg.Timestamp.Format("15:04:05"))
	}

	// Add metadata badges
	var badges []string

	// Sidechain indicator
	if msg.IsSidechain {
		badges = append(badges, "⑂sidechain")
	}

	// Model name (for assistant messages)
	if msg.Model != "" {
		badges = append(badges, formatModelName(msg.Model))
	}

	// Thinking level (for user messages that set it)
	if msg.ThinkingLevel != "" {
		badges = append(badges, "🧠"+msg.ThinkingLevel)
	}

	// Stop reason (for assistant messages, skip "end_turn" as it's the normal case)
	if msg.StopReason != "" && msg.StopReason != "end_turn" {
		badges = append(badges, "⏹"+msg.StopReason)
	}

	if len(badges) > 0 {
		header += " " + strings.Join(badges, " ")
	}

	// Show expand indicator
	expandIndicator := "▶"
	if isExpanded {
		expandIndicator = "▼"
	}
	header = expandIndicator + " " + header

	sb.WriteString(headerStyle.Render(header))
	sb.WriteString("\n")

	// Token usage line (for assistant messages with usage data)
	if msg.OutputTokens > 0 || msg.InputTokens > 0 || msg.CacheReadTokens > 0 {
		tokenLine := formatTokenUsage(msg)
		sb.WriteString("   " + TokenStyle.Render(tokenLine) + "\n")
	}

	// Show content preview or full content
	if isExpanded {
		// Show all blocks
		for _, block := range msg.Blocks {
			sb.WriteString(renderBlockFull(&block, maxWidth))
			sb.WriteString("\n")
		}
	} else {
		// Show preview
		preview := getMessagePreview(msg, maxWidth-4)
		sb.WriteString("   " + preview)
	}

	// Highlight if selected
	result := sb.String()
	if isSelected {
		// Add selection background to each line
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			lines[i] = SelectedMessageStyle.Render(line + strings.Repeat(" ", max(0, maxWidth-lipgloss.Width(line))))
		}
		result = strings.Join(lines, "\n")
	}

	return result
}

func renderBlockFull(b *data.ContentBlock, maxWidth int) string {
	var sb strings.Builder
	indent := "   "

	switch b.Type {
	case "thinking":
		sb.WriteString(indent + ThinkingStyle.Render("💭 [thinking]"))
		sb.WriteString("\n")
		for _, line := range strings.Split(b.Thinking, "\n") {
			wrapped := wrapLine(line, maxWidth-6)
			for _, wl := range wrapped {
				sb.WriteString(indent + ThinkingStyle.Render(wl) + "\n")
			}
		}

	case "text":
		// Wrap and display text
		wrapped := wrapText(b.Text, maxWidth-4)
		for _, line := range wrapped {
			sb.WriteString(indent + line + "\n")
		}

	case "tool_use":
		sb.WriteString(indent + ToolNameStyle.Render("🔧 "+b.ToolName))
		sb.WriteString("\n")
		if b.ToolInput != "" {
			for _, line := range strings.Split(b.ToolInput, "\n") {
				wrapped := wrapLine(line, maxWidth-6)
				for _, wl := range wrapped {
					sb.WriteString(indent + wl + "\n")
				}
			}
		}

	case "tool_result":
		sb.WriteString(indent + "📤 [tool result]")
		sb.WriteString("\n")
		if b.Result != "" {
			for _, line := range strings.Split(b.Result, "\n") {
				wrapped := wrapLine(line, maxWidth-6)
				for _, wl := range wrapped {
					sb.WriteString(indent + wl + "\n")
				}
			}
		}
	}

	return sb.String()
}

func getMessagePreview(msg *data.Message, maxWidth int) string {
	for _, b := range msg.Blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				preview := strings.ReplaceAll(b.Text, "\n", " ")
				return truncateWidth(preview, maxWidth)
			}
		case "thinking":
			return ThinkingStyle.Render("[thinking...]")
		case "tool_use":
			return ToolNameStyle.Render("🔧 " + b.ToolName + "()")
		case "tool_result":
			return "[tool result]"
		}
	}
	return "(empty)"
}

func renderNodeInfo(m model.Model) string {
	var sb strings.Builder

	switch m.Selected.Type {
	case model.NodeProject:
		sb.WriteString(fmt.Sprintf("Project: %s\n", m.Selected.Label))
		sb.WriteString(fmt.Sprintf("Path: %s\n", m.Selected.ID))
		sb.WriteString(fmt.Sprintf("Sessions: %d\n", len(m.Selected.Children)))

	case model.NodeMessage:
		if m.Selected.Message != nil {
			msg := m.Selected.Message
			sb.WriteString(fmt.Sprintf("Type: %s\n", msg.Type))
			sb.WriteString(fmt.Sprintf("Time: %s\n\n", msg.Timestamp.Format("15:04:05")))

			for _, block := range msg.Blocks {
				sb.WriteString(renderBlockFull(&block, m.Width-m.TreeWidth-8))
				sb.WriteString("\n")
			}
		}

	case model.NodeBlock:
		if m.Selected.Block != nil {
			sb.WriteString(renderBlockFull(m.Selected.Block, m.Width-m.TreeWidth-8))
		}
	}

	return sb.String()
}

func truncateWidth(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return "..."
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	// Simple truncation - might cut multi-byte chars but good enough
	for len(s) > 0 && lipgloss.Width(s) > maxWidth-3 {
		s = s[:len(s)-1]
	}
	return s + "..."
}

// wrapLine wraps a single line at character boundaries for display width
func wrapLine(s string, width int) []string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return []string{s}
	}

	var lines []string
	runes := []rune(s)
	start := 0

	for start < len(runes) {
		end := start + 1
		for end <= len(runes) && lipgloss.Width(string(runes[start:end])) <= width {
			end++
		}
		end-- // back up to last position that fit
		if end <= start {
			end = start + 1 // at least one char
		}
		lines = append(lines, string(runes[start:end]))
		start = end
	}
	return lines
}

func wrapText(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}

	var lines []string
	for _, para := range strings.Split(s, "\n") {
		if para == "" {
			lines = append(lines, "")
			continue
		}
		words := strings.Fields(para)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		line := words[0]
		for _, word := range words[1:] {
			if lipgloss.Width(line+" "+word) <= width {
				line += " " + word
			} else {
				lines = append(lines, line)
				line = word
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// formatModelName converts full model ID to a compact display name
func formatModelName(model string) string {
	// claude-opus-4-5-20251101 -> opus-4.5
	// claude-sonnet-4-20250514 -> sonnet-4
	// claude-3-5-haiku-20241022 -> haiku-3.5
	model = strings.TrimPrefix(model, "claude-")

	// Handle different model name patterns
	if strings.HasPrefix(model, "opus-4-5") {
		return "opus-4.5"
	}
	if strings.HasPrefix(model, "opus-4-") || strings.HasPrefix(model, "opus-4") {
		return "opus-4"
	}
	if strings.HasPrefix(model, "sonnet-4-") || strings.HasPrefix(model, "sonnet-4") {
		return "sonnet-4"
	}
	if strings.HasPrefix(model, "3-5-sonnet") || strings.HasPrefix(model, "3-5-sonnet") {
		return "sonnet-3.5"
	}
	if strings.HasPrefix(model, "3-5-haiku") {
		return "haiku-3.5"
	}
	if strings.HasPrefix(model, "3-opus") {
		return "opus-3"
	}
	if strings.HasPrefix(model, "3-sonnet") {
		return "sonnet-3"
	}
	if strings.HasPrefix(model, "3-haiku") {
		return "haiku-3"
	}

	// Fallback: take first part before date suffix
	if idx := strings.LastIndex(model, "-20"); idx > 0 {
		return model[:idx]
	}
	return model
}

// formatTokenUsage creates compact token usage display
// Format: "Tokens: 12940 cached + 14703 uncached -> 3"
func formatTokenUsage(msg *data.Message) string {
	uncached := msg.InputTokens + msg.CacheWriteTokens
	cached := msg.CacheReadTokens
	out := msg.OutputTokens

	var parts []string
	if cached > 0 {
		parts = append(parts, fmt.Sprintf("%d cached", cached))
	}
	if uncached > 0 {
		parts = append(parts, fmt.Sprintf("%d uncached", uncached))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("→ %d tokens", out)
	}

	return fmt.Sprintf("Tokens: %s → %d", strings.Join(parts, " + "), out)
}
