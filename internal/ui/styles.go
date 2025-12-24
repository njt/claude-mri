package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	active    = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	// Layout
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle)

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(highlight).
			Padding(0, 1)

	// Tree pane
	TreePaneStyle = BorderStyle.Width(35)

	// Detail pane
	DetailPaneStyle = BorderStyle

	// Tree items
	TreeItemStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	SelectedStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			Background(highlight).
			Foreground(lipgloss.Color("#FFFFFF"))

	// Content blocks
	ThinkingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	ToolNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D")).
			Bold(true)

	// Status
	ActiveIndicator = lipgloss.NewStyle().
			Foreground(active).
			SetString("●")

	InactiveIndicator = lipgloss.NewStyle().
			Foreground(subtle).
			SetString("○")

	FollowOnStyle = lipgloss.NewStyle().
			Foreground(active).
			Bold(true)

	FollowOffStyle = lipgloss.NewStyle().
			Foreground(subtle)

	// Help
	HelpStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Padding(0, 1)
)
