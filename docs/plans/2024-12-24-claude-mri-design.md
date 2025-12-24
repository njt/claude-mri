# claude-mri Design

A TUI for seeing inside Claude's mind - browse projects, sessions, messages, thinking blocks, and tool calls with live monitoring.

## Overview

**claude-mri** lets you:
- Browse the hierarchy: projects → sessions → messages → thinking/tools
- Watch live activity as Claude and subagents work
- Inspect thinking blocks, tool inputs/outputs, conversation flow

## Decisions

| Aspect | Decision |
|--------|----------|
| Scope | Historical browsing + live monitoring |
| Framework | Bubble Tea (Go) - single binary |
| Layout | Dual pane: tree (left) + details (right) |
| Live mode | Configurable follow with `f` toggle |
| Thinking | Collapsed by default (5 lines), expandable |
| Tool calls | Inline with message flow |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ claude-mri                                      [F]ollow: ON    │
├────────────────────────┬────────────────────────────────────────┤
│ ▼ Projects             │ Message Details                        │
│   ▼ claude-mri         │ ─────────────────────────────────────  │
│     ▼ Session 52c48... │ [thinking] ▶ The user wants me to...   │
│       ├─ User: "Use...│                                        │
│       ├─ Asst: Tool... │ [text] This is a creative design...    │
│       │  └─ Skill(...) │                                        │
│       └─ User: "Base.. │ [tool_use] Skill                       │
│     ○ Session abc12... │   skill: "superpowers:brainstorming"   │
│   ▶ beads              │                                        │
│   ▶ Encarta            │ [tool_result] Launching skill...       │
│                        │                                        │
├────────────────────────┴────────────────────────────────────────┤
│ j/k:navigate  Enter:expand  f:follow  q:quit  /:search          │
└─────────────────────────────────────────────────────────────────┘
```

### Core Components

- **Data layer**: Watches `~/.claude/projects/` and parses JSONL files
- **Model**: Elm-architecture state (projects → sessions → messages tree)
- **View**: Dual-pane with tree (left) and details (right)
- **Live mode**: fsnotify watches for file changes, streams new entries

### Data Hierarchy

```
Project (folder name)
 └─ Session (uuid.jsonl)
     └─ Message (user/assistant)
         ├─ Thinking block
         ├─ Text content
         ├─ Tool use → Tool result
         └─ Subagent (agent-xxx.jsonl, linked by agentId)
```

## Data Model

### JSONL Structure

Claude stores data in `~/.claude/projects/{project-path}/`:
- `{uuid}.jsonl` - main session files
- `agent-{id}.jsonl` - subagent conversations

### Go Types

```go
type Message struct {
    UUID        string    `json:"uuid"`
    ParentUUID  *string   `json:"parentUuid"`
    Type        string    `json:"type"`      // "user" | "assistant"
    Timestamp   time.Time `json:"timestamp"`
    SessionID   string    `json:"sessionId"`
    AgentID     *string   `json:"agentId"`   // present for subagents
    IsSidechain bool      `json:"isSidechain"`
    Message     Content   `json:"message"`
}

type Content struct {
    Role    string        `json:"role"`
    Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
    Type      string `json:"type"`  // "thinking" | "text" | "tool_use" | "tool_result"
    Text      string `json:"text,omitempty"`
    Thinking  string `json:"thinking,omitempty"`
    Name      string `json:"name,omitempty"`      // tool name
    Input     any    `json:"input,omitempty"`     // tool input
    ToolUseID string `json:"tool_use_id,omitempty"`
    Content   any    `json:"content,omitempty"`   // tool result
}
```

### Parsing Strategy

- On startup: scan `~/.claude/projects/*/` for all `.jsonl` files
- Parse each line as JSON, build in-memory tree
- Link messages via `parentUuid` to reconstruct conversation flow
- Link subagent files via `agentId` field
- Skip `file-history-snapshot` entries

## UI Components

### Left Pane - Tree View

- Expandable/collapsible nodes with `▶`/`▼` indicators
- Projects: decoded folder name (e.g., `C--Users-Nat-source-beads` → `beads`)
- Sessions: truncated UUID + timestamp
- Messages: type icon + preview
  - `👤 "Use your superpowers..."` (user)
  - `🤖 "This is a creative..."` (assistant)
  - `🔧 Skill(brainstorming)` (tool use)
  - `💭 ▶ thinking...` (collapsed thinking)
- Subagents nested under spawning message
- Active sessions marked with `●`

### Right Pane - Details

- Full content of selected node
- Thinking: first 5 lines + `[+N more lines]` expand hint
- Tool inputs: syntax-highlighted JSON
- Tool results: formatted, truncated if large
- Relative timestamps in live mode

### Keybindings

| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate tree |
| `Enter` or `→` | Expand node / show details |
| `Esc` or `←` | Collapse / go back |
| `f` | Toggle follow mode |
| `/` | Search (filters tree) |
| `q` | Quit |
| `Space` | Expand collapsed content |

## Live Monitoring

### File Watching

1. `fsnotify` monitors `~/.claude/projects/` recursively
2. Track byte offset per file, parse only new lines
3. New `.jsonl` files = new sessions
4. Link `agent-xxx.jsonl` to parent via `sessionId`

### Follow Mode

- **ON**: Auto-expand to latest, update details, highlight new items
- **OFF**: Badge shows `(N new)`, selection stays put, `f` jumps to latest

### Visual Indicators

```
● Session 52c48...     ← green = active (modified < 30s ago)
○ Session abc12...     ← hollow = inactive
```

### Edge Cases

- Partial JSON lines: skip until newline complete
- Deleted sessions: mark as ended, keep in tree
- Large files: lazy-load, keep recent N in memory
- Rapid updates: debounce to 100ms

## Project Structure

```
claude-mri/
├── main.go              # Entry point, CLI flags
├── internal/
│   ├── data/
│   │   ├── parser.go    # JSONL parsing
│   │   ├── watcher.go   # fsnotify file watching
│   │   └── types.go     # Message, Session, Project structs
│   ├── model/
│   │   ├── model.go     # Bubble Tea model, state
│   │   ├── tree.go      # Tree node operations
│   │   └── update.go    # Message handling
│   └── ui/
│       ├── view.go      # Main render function
│       ├── tree.go      # Left pane tree component
│       ├── details.go   # Right pane details component
│       ├── styles.go    # Lipgloss styles
│       └── keys.go      # Keybinding definitions
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - Viewport, textinput
- `github.com/fsnotify/fsnotify` - File watching

## CLI Usage

```bash
claude-mri                    # Watch default ~/.claude/projects
claude-mri --path /other/dir  # Custom path
claude-mri --no-live          # Historical only, no watching
```

## Distribution

- Single binary, no runtime dependencies
- Cross-compile: `GOOS=windows/darwin/linux GOARCH=amd64/arm64`
