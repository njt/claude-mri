# claude-mri Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a TUI to browse Claude's internal state (projects, sessions, messages, thinking, tools) with live monitoring.

**Architecture:** Bubble Tea (Elm architecture) with dual-pane layout. Data layer parses JSONL from ~/.claude/projects/. File watcher enables live updates.

**Tech Stack:** Go, Bubble Tea, Lipgloss, Bubbles, fsnotify

---

## Task 1: Project Setup

**Files:**
- Create: `go.mod`
- Create: `main.go`

**Step 1: Initialize Go module**

Run:
```bash
cd C:/Users/Nat/source/claude-mri
go mod init github.com/natdempk/claude-mri
```

**Step 2: Add dependencies**

Run:
```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go get github.com/charmbracelet/bubbles/viewport
go get github.com/charmbracelet/bubbles/key
go get github.com/fsnotify/fsnotify
```

**Step 3: Create minimal main.go**

```go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	message string
}

func initialModel() model {
	return model{message: "claude-mri starting..."}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	return m.message + "\n\nPress q to quit."
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

**Step 4: Verify it runs**

Run:
```bash
go run main.go
```
Expected: TUI appears with "claude-mri starting...", press q to quit.

**Step 5: Commit**

```bash
git init
git add .
git commit -m "feat: initial project setup with bubble tea skeleton"
```

---

## Task 2: Data Types

**Files:**
- Create: `internal/data/types.go`

**Step 1: Create internal/data directory**

Run:
```bash
mkdir -p internal/data
```

**Step 2: Write data types**

```go
package data

import (
	"encoding/json"
	"time"
)

// Project represents a Claude project folder
type Project struct {
	Name     string
	Path     string
	Sessions []*Session
}

// Session represents a conversation session (uuid.jsonl file)
type Session struct {
	ID        string
	FilePath  string
	Messages  []*Message
	IsAgent   bool      // true for agent-xxx.jsonl files
	AgentID   string    // populated for agent files
	UpdatedAt time.Time
}

// Message represents a single JSONL entry
type Message struct {
	UUID        string     `json:"uuid"`
	ParentUUID  *string    `json:"parentUuid"`
	Type        string     `json:"type"` // "user" | "assistant"
	Timestamp   time.Time  `json:"timestamp"`
	SessionID   string     `json:"sessionId"`
	AgentID     *string    `json:"agentId"`
	IsSidechain bool       `json:"isSidechain"`
	Message     RawContent `json:"message"`
	// Parsed content blocks
	Blocks []ContentBlock `json:"-"`
}

// RawContent holds the raw message content from JSON
type RawContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"` // can be string or array
}

// ContentBlock represents a parsed content block
type ContentBlock struct {
	Type      string // "thinking" | "text" | "tool_use" | "tool_result"
	Text      string // for text blocks
	Thinking  string // for thinking blocks
	ToolName  string // for tool_use
	ToolInput string // JSON string of input
	ToolID    string // tool_use_id
	Result    string // for tool_result
}

// TreeNode is the interface for displayable tree items
type TreeNode interface {
	NodeID() string
	NodeLabel() string
	NodeChildren() []TreeNode
	IsExpandable() bool
}
```

**Step 3: Verify it compiles**

Run:
```bash
go build ./...
```
Expected: No errors

**Step 4: Commit**

```bash
git add internal/
git commit -m "feat: add data types for projects, sessions, messages"
```

---

## Task 3: JSONL Parser

**Files:**
- Create: `internal/data/parser.go`
- Create: `internal/data/parser_test.go`

**Step 1: Write parser test**

```go
package data

import (
	"testing"
)

func TestParseMessageLine_UserMessage(t *testing.T) {
	line := `{"type":"user","uuid":"abc123","timestamp":"2025-12-22T22:20:39.768Z","sessionId":"session1","message":{"role":"user","content":"Hello"}}`

	msg, err := ParseMessageLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Type != "user" {
		t.Errorf("expected type 'user', got %q", msg.Type)
	}
	if msg.UUID != "abc123" {
		t.Errorf("expected uuid 'abc123', got %q", msg.UUID)
	}
	if len(msg.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(msg.Blocks))
	}
	if msg.Blocks[0].Type != "text" {
		t.Errorf("expected block type 'text', got %q", msg.Blocks[0].Type)
	}
	if msg.Blocks[0].Text != "Hello" {
		t.Errorf("expected text 'Hello', got %q", msg.Blocks[0].Text)
	}
}

func TestParseMessageLine_AssistantWithThinking(t *testing.T) {
	line := `{"type":"assistant","uuid":"def456","timestamp":"2025-12-22T22:20:47.983Z","sessionId":"session1","message":{"role":"assistant","content":[{"type":"thinking","thinking":"Let me think..."},{"type":"text","text":"Here is my response"}]}}`

	msg, err := ParseMessageLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(msg.Blocks))
	}
	if msg.Blocks[0].Type != "thinking" {
		t.Errorf("expected block 0 type 'thinking', got %q", msg.Blocks[0].Type)
	}
	if msg.Blocks[0].Thinking != "Let me think..." {
		t.Errorf("expected thinking text, got %q", msg.Blocks[0].Thinking)
	}
	if msg.Blocks[1].Type != "text" {
		t.Errorf("expected block 1 type 'text', got %q", msg.Blocks[1].Type)
	}
}

func TestParseMessageLine_ToolUse(t *testing.T) {
	line := `{"type":"assistant","uuid":"ghi789","timestamp":"2025-12-22T22:20:49.806Z","sessionId":"session1","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Bash","input":{"command":"ls"}}]}}`

	msg, err := ParseMessageLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msg.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(msg.Blocks))
	}
	if msg.Blocks[0].Type != "tool_use" {
		t.Errorf("expected type 'tool_use', got %q", msg.Blocks[0].Type)
	}
	if msg.Blocks[0].ToolName != "Bash" {
		t.Errorf("expected tool name 'Bash', got %q", msg.Blocks[0].ToolName)
	}
}

func TestParseMessageLine_SkipsSnapshot(t *testing.T) {
	line := `{"type":"file-history-snapshot","messageId":"abc"}`

	msg, err := ParseMessageLine([]byte(line))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != nil {
		t.Errorf("expected nil for snapshot, got message")
	}
}
```

**Step 2: Run test to verify it fails**

Run:
```bash
go test ./internal/data/... -v
```
Expected: FAIL - ParseMessageLine not defined

**Step 3: Write parser implementation**

```go
package data

import (
	"encoding/json"
	"strings"
)

// ParseMessageLine parses a single JSONL line into a Message
// Returns nil for non-message entries (snapshots, etc.)
func ParseMessageLine(line []byte) (*Message, error) {
	// Quick check for message type
	var typeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(line, &typeCheck); err != nil {
		return nil, err
	}

	// Skip non-message types
	if typeCheck.Type != "user" && typeCheck.Type != "assistant" {
		return nil, nil
	}

	var msg Message
	if err := json.Unmarshal(line, &msg); err != nil {
		return nil, err
	}

	// Parse content blocks
	msg.Blocks = parseContentBlocks(msg.Message.Content)

	return &msg, nil
}

// parseContentBlocks handles both string and array content
func parseContentBlocks(raw json.RawMessage) []ContentBlock {
	if len(raw) == 0 {
		return nil
	}

	// Try as string first
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return []ContentBlock{{Type: "text", Text: str}}
	}

	// Try as array
	var blocks []struct {
		Type     string          `json:"type"`
		Text     string          `json:"text,omitempty"`
		Thinking string          `json:"thinking,omitempty"`
		Name     string          `json:"name,omitempty"`
		ID       string          `json:"id,omitempty"`
		Input    json.RawMessage `json:"input,omitempty"`
		Content  json.RawMessage `json:"content,omitempty"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil
	}

	result := make([]ContentBlock, 0, len(blocks))
	for _, b := range blocks {
		block := ContentBlock{Type: b.Type}
		switch b.Type {
		case "text":
			block.Text = b.Text
		case "thinking":
			block.Thinking = b.Thinking
		case "tool_use":
			block.ToolName = b.Name
			block.ToolID = b.ID
			if len(b.Input) > 0 {
				block.ToolInput = formatJSON(b.Input)
			}
		case "tool_result":
			block.ToolID = b.ID
			if len(b.Content) > 0 {
				block.Result = formatJSON(b.Content)
			}
		}
		result = append(result, block)
	}
	return result
}

// formatJSON formats JSON for display
func formatJSON(raw json.RawMessage) string {
	// Try to pretty print
	var v interface{}
	if err := json.Unmarshal(raw, &v); err == nil {
		if pretty, err := json.MarshalIndent(v, "", "  "); err == nil {
			return string(pretty)
		}
	}
	return strings.TrimSpace(string(raw))
}
```

**Step 4: Run tests to verify they pass**

Run:
```bash
go test ./internal/data/... -v
```
Expected: PASS

**Step 5: Commit**

```bash
git add internal/data/
git commit -m "feat: add JSONL parser with tests"
```

---

## Task 4: Project Scanner

**Files:**
- Create: `internal/data/scanner.go`

**Step 1: Write scanner**

```go
package data

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	// Matches session files: uuid.jsonl
	sessionFileRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.jsonl$`)
	// Matches agent files: agent-xxx.jsonl
	agentFileRe = regexp.MustCompile(`^agent-([a-f0-9]+)\.jsonl$`)
)

// ScanProjects scans the Claude projects directory and returns all projects
func ScanProjects(basePath string) ([]*Project, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, err
	}

	var projects []*Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projPath := filepath.Join(basePath, entry.Name())
		proj := &Project{
			Name: decodeProjectName(entry.Name()),
			Path: projPath,
		}

		// Scan sessions
		sessions, err := scanSessions(projPath)
		if err != nil {
			continue // skip projects we can't read
		}
		proj.Sessions = sessions
		projects = append(projects, proj)
	}

	// Sort projects by name
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	return projects, nil
}

// decodeProjectName converts folder name to readable project name
// e.g., "C--Users-Nat-source-beads" -> "beads"
func decodeProjectName(name string) string {
	// Take the last path segment
	parts := strings.Split(name, "-")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return name
}

// scanSessions finds all session files in a project directory
func scanSessions(projPath string) ([]*Session, error) {
	entries, err := os.ReadDir(projPath)
	if err != nil {
		return nil, err
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		filePath := filepath.Join(projPath, name)

		var session *Session
		if sessionFileRe.MatchString(name) {
			// Main session file
			id := strings.TrimSuffix(name, ".jsonl")
			session = &Session{
				ID:       id,
				FilePath: filePath,
				IsAgent:  false,
			}
		} else if matches := agentFileRe.FindStringSubmatch(name); matches != nil {
			// Agent file
			session = &Session{
				ID:       matches[1],
				FilePath: filePath,
				IsAgent:  true,
				AgentID:  matches[1],
			}
		}

		if session != nil {
			info, _ := entry.Info()
			if info != nil {
				session.UpdatedAt = info.ModTime()
			}
			sessions = append(sessions, session)
		}
	}

	// Sort by update time, newest first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions, nil
}

// LoadSession loads all messages from a session file
func LoadSession(session *Session) error {
	file, err := os.Open(session.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		msg, err := ParseMessageLine(scanner.Bytes())
		if err != nil {
			continue // skip malformed lines
		}
		if msg != nil {
			session.Messages = append(session.Messages, msg)
		}
	}

	return scanner.Err()
}
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./...
```
Expected: No errors

**Step 3: Commit**

```bash
git add internal/data/
git commit -m "feat: add project/session scanner"
```

---

## Task 5: Tree Model

**Files:**
- Create: `internal/model/tree.go`

**Step 1: Create model directory and tree**

Run:
```bash
mkdir -p internal/model
```

**Step 2: Write tree model**

```go
package model

import "github.com/natdempk/claude-mri/internal/data"

// NodeType identifies the type of tree node
type NodeType int

const (
	NodeProject NodeType = iota
	NodeSession
	NodeMessage
	NodeBlock
)

// TreeNode represents a node in the navigation tree
type TreeNode struct {
	Type     NodeType
	ID       string
	Label    string
	Expanded bool
	Children []*TreeNode
	// References to underlying data
	Project *data.Project
	Session *data.Session
	Message *data.Message
	Block   *data.ContentBlock
}

// BuildTree creates the tree structure from projects
func BuildTree(projects []*data.Project) []*TreeNode {
	nodes := make([]*TreeNode, 0, len(projects))
	for _, p := range projects {
		nodes = append(nodes, buildProjectNode(p))
	}
	return nodes
}

func buildProjectNode(p *data.Project) *TreeNode {
	node := &TreeNode{
		Type:     NodeProject,
		ID:       p.Path,
		Label:    p.Name,
		Expanded: false,
		Project:  p,
	}
	for _, s := range p.Sessions {
		node.Children = append(node.Children, buildSessionNode(s))
	}
	return node
}

func buildSessionNode(s *data.Session) *TreeNode {
	label := s.ID[:8] + "..."
	if s.IsAgent {
		label = "agent-" + s.AgentID
	}
	node := &TreeNode{
		Type:     NodeSession,
		ID:       s.FilePath,
		Label:    label,
		Expanded: false,
		Session:  s,
	}
	for _, m := range s.Messages {
		node.Children = append(node.Children, buildMessageNode(m))
	}
	return node
}

func buildMessageNode(m *data.Message) *TreeNode {
	icon := "👤"
	if m.Type == "assistant" {
		icon = "🤖"
	}
	label := icon + " " + truncate(getMessagePreview(m), 30)

	node := &TreeNode{
		Type:     NodeMessage,
		ID:       m.UUID,
		Label:    label,
		Expanded: false,
		Message:  m,
	}

	// Add blocks as children for detailed view
	for i := range m.Blocks {
		b := &m.Blocks[i]
		node.Children = append(node.Children, buildBlockNode(b))
	}
	return node
}

func buildBlockNode(b *data.ContentBlock) *TreeNode {
	var label string
	switch b.Type {
	case "thinking":
		label = "💭 thinking..."
	case "text":
		label = "📝 " + truncate(b.Text, 25)
	case "tool_use":
		label = "🔧 " + b.ToolName
	case "tool_result":
		label = "📤 result"
	default:
		label = b.Type
	}
	return &TreeNode{
		Type:  NodeBlock,
		ID:    b.ToolID,
		Label: label,
		Block: b,
	}
}

func getMessagePreview(m *data.Message) string {
	for _, b := range m.Blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				return b.Text
			}
		case "tool_use":
			return b.ToolName + "()"
		}
	}
	return "(empty)"
}

func truncate(s string, max int) string {
	// Remove newlines
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
```

**Step 3: Add missing import**

Add to imports in tree.go:
```go
import (
	"strings"

	"github.com/natdempk/claude-mri/internal/data"
)
```

**Step 4: Verify it compiles**

Run:
```bash
go build ./...
```
Expected: No errors

**Step 5: Commit**

```bash
git add internal/model/
git commit -m "feat: add tree model for navigation"
```

---

## Task 6: Styles

**Files:**
- Create: `internal/ui/styles.go`

**Step 1: Create ui directory and styles**

Run:
```bash
mkdir -p internal/ui
```

**Step 2: Write styles**

```go
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
```

**Step 3: Verify it compiles**

Run:
```bash
go build ./...
```
Expected: No errors

**Step 4: Commit**

```bash
git add internal/ui/
git commit -m "feat: add UI styles"
```

---

## Task 7: Main Model

**Files:**
- Modify: `main.go`
- Create: `internal/model/model.go`

**Step 1: Write the main model**

```go
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
	Cursor    int
	Selected  *TreeNode

	// UI state
	FollowMode  bool
	Ready       bool
	Width       int
	Height      int
	TreeWidth   int
	DetailView  viewport.Model

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
```

**Step 2: Write Update function**

Create `internal/model/update.go`:

```go
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

	case errMsg:
		// Could display error, for now just ignore
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
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
						m.Selected.Children = append(m.Selected.Children, buildMessageNode(msg))
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
```

**Step 3: Add depth field to TreeNode**

Update `internal/model/tree.go`, add to TreeNode struct:
```go
type TreeNode struct {
	// ... existing fields ...
	depth int // for rendering indentation
}

// Depth returns the node's depth in the tree
func (n *TreeNode) Depth() int {
	return n.depth
}
```

**Step 4: Verify it compiles**

Run:
```bash
go build ./...
```
Expected: No errors

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add main model with navigation"
```

---

## Task 8: View Rendering

**Files:**
- Create: `internal/ui/view.go`
- Modify: `main.go`

**Step 1: Write view rendering**

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

	for i, node := range m.FlatNodes {
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

func renderBlock(b *model.ContentBlock) string {
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
```

**Step 2: Fix ContentBlock reference**

The renderBlock function uses `model.ContentBlock` but it should be `data.ContentBlock`. Update imports and fix:

```go
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/natdempk/claude-mri/internal/data"
	"github.com/natdempk/claude-mri/internal/model"
)

// ... in renderBlock signature:
func renderBlock(b *data.ContentBlock) string {
```

**Step 3: Update main.go to use our model**

```go
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
```

**Step 4: Verify it compiles and runs**

Run:
```bash
go build ./...
go run .
```
Expected: TUI appears with project tree

**Step 5: Commit**

```bash
git add .
git commit -m "feat: add view rendering with tree and details panes"
```

---

## Task 9: File Watcher for Live Mode

**Files:**
- Create: `internal/data/watcher.go`

**Step 1: Write file watcher**

```go
package data

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// FileEvent represents a file change
type FileEvent struct {
	Path    string
	Project string
	IsNew   bool
}

// Watcher watches the Claude projects directory for changes
type Watcher struct {
	watcher  *fsnotify.Watcher
	basePath string
	Events   chan FileEvent
	Errors   chan error
	done     chan struct{}
}

// NewWatcher creates a new file watcher
func NewWatcher(basePath string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher:  w,
		basePath: basePath,
		Events:   make(chan FileEvent, 100),
		Errors:   make(chan error, 10),
		done:     make(chan struct{}),
	}, nil
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	// Watch base path
	if err := w.watcher.Add(w.basePath); err != nil {
		return err
	}

	// Watch all project directories
	projects, err := ScanProjects(w.basePath)
	if err != nil {
		return err
	}
	for _, p := range projects {
		if err := w.watcher.Add(p.Path); err != nil {
			log.Printf("Warning: could not watch %s: %v", p.Path, err)
		}
	}

	go w.run()
	return nil
}

func (w *Watcher) run() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Only care about writes and creates to .jsonl files
			if !strings.HasSuffix(event.Name, ".jsonl") {
				continue
			}

			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Extract project name from path
			rel, _ := filepath.Rel(w.basePath, event.Name)
			parts := strings.Split(rel, string(filepath.Separator))
			project := ""
			if len(parts) > 0 {
				project = parts[0]
			}

			w.Events <- FileEvent{
				Path:    event.Name,
				Project: project,
				IsNew:   event.Op&fsnotify.Create != 0,
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.Errors <- err

		case <-w.done:
			return
		}
	}
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
}
```

**Step 2: Verify it compiles**

Run:
```bash
go build ./...
```
Expected: No errors

**Step 3: Commit**

```bash
git add internal/data/watcher.go
git commit -m "feat: add file watcher for live mode"
```

---

## Task 10: Integrate Watcher into Model

**Files:**
- Modify: `internal/model/model.go`
- Modify: `internal/model/update.go`

**Step 1: Add watcher to model**

Add to `internal/model/model.go`:

```go
// Add to Model struct:
Watcher *data.Watcher

// Add message type:
type fileEventMsg data.FileEvent

// Update NewModel:
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

// Update Init:
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

// Add watch command:
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
```

**Step 2: Handle file events in update**

Add to `internal/model/update.go` in the Update switch:

```go
case fileEventMsg:
	// Reload projects on file change
	// In follow mode, this will auto-update the tree
	return m, tea.Batch(m.loadProjects, m.watchFiles)
```

**Step 3: Verify it compiles and runs**

Run:
```bash
go build ./...
go run .
```
Expected: TUI updates when Claude sessions change

**Step 4: Commit**

```bash
git add internal/model/
git commit -m "feat: integrate file watcher for live updates"
```

---

## Task 11: Polish and README

**Files:**
- Create: `README.md`

**Step 1: Write README**

```markdown
# claude-mri

A TUI for seeing inside Claude's mind - browse projects, sessions, messages, thinking blocks, and tool calls with live monitoring.

## Features

- Browse the hierarchy: projects → sessions → messages → thinking/tools
- Watch live activity as Claude and subagents work
- Inspect thinking blocks, tool inputs/outputs, conversation flow
- Vim-style keyboard navigation

## Installation

```bash
go install github.com/natdempk/claude-mri@latest
```

Or build from source:

```bash
git clone https://github.com/natdempk/claude-mri
cd claude-mri
go build
```

## Usage

```bash
claude-mri                    # Watch default ~/.claude/projects
claude-mri --path /other/dir  # Custom path
```

## Keybindings

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate tree |
| `Enter` or `→` | Expand node |
| `Esc` or `←` | Collapse node |
| `f` | Toggle follow mode |
| `q` | Quit |

## License

MIT
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README"
```

---

## Summary

The implementation is broken into 11 tasks:

1. **Project Setup** - Go module, dependencies, skeleton
2. **Data Types** - Structs for projects, sessions, messages
3. **JSONL Parser** - Parse Claude's data format (TDD)
4. **Project Scanner** - Find and load sessions
5. **Tree Model** - Navigation tree structure
6. **Styles** - Lipgloss styling
7. **Main Model** - Bubble Tea state management
8. **View Rendering** - Dual-pane UI
9. **File Watcher** - fsnotify for live updates
10. **Watcher Integration** - Connect watcher to model
11. **Polish** - README and cleanup

Each task builds on the previous, with tests where appropriate and commits after each step.
