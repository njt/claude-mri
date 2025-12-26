package model

import (
	"strings"

	"github.com/natdempk/claude-mri/internal/data"
)

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
	// for rendering indentation
	depth int
}

// Depth returns the node's depth in the tree
func (n *TreeNode) Depth() int {
	return n.depth
}

// IsExpandable returns true if the node can be expanded
// Sessions and messages can always be expanded (lazy-load children)
func (n *TreeNode) IsExpandable() bool {
	switch n.Type {
	case NodeSession, NodeMessage:
		return true // Can always expand, children loaded lazily
	default:
		return len(n.Children) > 0
	}
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
	label := s.ID
	if len(s.ID) > 8 {
		label = s.ID[:8] + "..."
	}
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
		node.Children = append(node.Children, BuildMessageNode(m))
	}
	return node
}

func BuildMessageNode(m *data.Message) *TreeNode {
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
