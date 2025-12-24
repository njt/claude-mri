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
