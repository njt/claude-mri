package data

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/natdempk/claude-mri/internal/debug"
)

var (
	// Matches session files: uuid.jsonl
	sessionFileRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.jsonl$`)
	// Matches agent files: agent-xxx.jsonl
	agentFileRe = regexp.MustCompile(`^agent-([a-f0-9]+)\.jsonl$`)
)

// ScanProjects scans the Claude projects directory and returns all projects
func ScanProjects(basePath string) ([]*Project, error) {
	defer debug.Time("ScanProjects")()

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

	debug.Log("ScanProjects found %d projects", len(projects))
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
	defer debug.Time("LoadSession " + session.ID)()

	file, err := os.Open(session.FilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Increase buffer for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	lineCount := 0
	for scanner.Scan() {
		lineCount++
		msg, err := ParseMessageLine(scanner.Bytes())
		if err != nil {
			continue // skip malformed lines
		}
		if msg != nil {
			session.Messages = append(session.Messages, msg)
		}
	}

	debug.Log("LoadSession parsed %d lines, %d messages", lineCount, len(session.Messages))
	return scanner.Err()
}
