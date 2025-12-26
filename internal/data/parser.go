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

	// Parse additional metadata from the raw JSON
	parseMessageMetadata(line, &msg)

	return &msg, nil
}

// parseMessageMetadata extracts model, usage, stop_reason, and thinking level
func parseMessageMetadata(line []byte, msg *Message) {
	// Parse the full structure to get nested fields
	var raw struct {
		ThinkingMetadata *struct {
			Level string `json:"level"`
		} `json:"thinkingMetadata"`
		Message *struct {
			Model      string `json:"model"`
			StopReason string `json:"stop_reason"`
			Usage      *struct {
				InputTokens             int `json:"input_tokens"`
				OutputTokens            int `json:"output_tokens"`
				CacheReadInputTokens    int `json:"cache_read_input_tokens"`
				CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}

	if err := json.Unmarshal(line, &raw); err != nil {
		return
	}

	// Extract thinking level (from user messages)
	if raw.ThinkingMetadata != nil {
		msg.ThinkingLevel = raw.ThinkingMetadata.Level
	}

	// Extract model, stop_reason, and usage (from assistant messages)
	if raw.Message != nil {
		msg.Model = raw.Message.Model
		msg.StopReason = raw.Message.StopReason

		if raw.Message.Usage != nil {
			msg.InputTokens = raw.Message.Usage.InputTokens
			msg.OutputTokens = raw.Message.Usage.OutputTokens
			msg.CacheReadTokens = raw.Message.Usage.CacheReadInputTokens
			msg.CacheWriteTokens = raw.Message.Usage.CacheCreationInputTokens
		}
	}
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
