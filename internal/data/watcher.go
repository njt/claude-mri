package data

import (
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

	// Debouncing
	debounceDelay time.Duration
	pending       map[string]FileEvent
	pendingMu     sync.Mutex
	debounceTimer *time.Timer
}

// NewWatcher creates a new file watcher
func NewWatcher(basePath string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		watcher:       w,
		basePath:      basePath,
		Events:        make(chan FileEvent, 100),
		Errors:        make(chan error, 10),
		done:          make(chan struct{}),
		debounceDelay: 500 * time.Millisecond,
		pending:       make(map[string]FileEvent),
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

			// Debounce: add to pending and reset timer
			w.pendingMu.Lock()
			w.pending[event.Name] = FileEvent{
				Path:    event.Name,
				Project: project,
				IsNew:   event.Op&fsnotify.Create != 0,
			}

			// Reset or start debounce timer
			if w.debounceTimer != nil {
				w.debounceTimer.Stop()
			}
			w.debounceTimer = time.AfterFunc(w.debounceDelay, w.flushPending)
			w.pendingMu.Unlock()

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

// flushPending sends all pending events as a single event
func (w *Watcher) flushPending() {
	w.pendingMu.Lock()
	defer w.pendingMu.Unlock()

	if len(w.pending) == 0 {
		return
	}

	// Send just one event (the most recent one) to trigger a refresh
	var lastEvent FileEvent
	for _, evt := range w.pending {
		lastEvent = evt
	}

	// Clear pending
	w.pending = make(map[string]FileEvent)

	// Send the event
	select {
	case w.Events <- lastEvent:
	default:
		// Channel full, skip
	}
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
	w.watcher.Close()
}
