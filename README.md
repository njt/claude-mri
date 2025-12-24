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
