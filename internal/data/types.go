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

// MostRecentUpdate returns the most recent session update time for this project
func (p *Project) MostRecentUpdate() time.Time {
	var most time.Time
	for _, s := range p.Sessions {
		if s.UpdatedAt.After(most) {
			most = s.UpdatedAt
		}
	}
	return most
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

	// Parsed metadata (extracted from nested JSON)
	Model         string `json:"-"` // e.g., "claude-opus-4-5-20251101"
	StopReason    string `json:"-"` // e.g., "end_turn", "tool_use", "max_tokens"
	ThinkingLevel string `json:"-"` // e.g., "high", "low", ""

	// Token usage
	InputTokens      int `json:"-"`
	OutputTokens     int `json:"-"`
	CacheReadTokens  int `json:"-"`
	CacheWriteTokens int `json:"-"`
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
