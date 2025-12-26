package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	subtle     = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight  = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	active     = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	userColor  = lipgloss.AdaptiveColor{Light: "#1E88E5", Dark: "#64B5F6"}
	assistantColor = lipgloss.AdaptiveColor{Light: "#7B1FA2", Dark: "#CE93D8"}

	// Layout
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle)

	FocusedBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(highlight)

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(highlight).
			Padding(0, 1)

	// Tree pane
	TreePaneStyle = BorderStyle.Width(40)
	TreePaneFocusedStyle = FocusedBorderStyle.Width(40)

	// Detail pane
	DetailPaneStyle = BorderStyle
	DetailPaneFocusedStyle = FocusedBorderStyle

	// Message styles
	UserMessageStyle = lipgloss.NewStyle().
			Foreground(userColor).
			Bold(true)

	AssistantMessageStyle = lipgloss.NewStyle().
			Foreground(assistantColor).
			Bold(true)

	SelectedMessageStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#3A3A3A"))

	MessageHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(1)

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

	TokenStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

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
