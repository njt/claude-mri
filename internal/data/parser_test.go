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
