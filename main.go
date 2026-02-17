package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cfg := LoadConfig()
	m := initialModel(cfg, root)

	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start file watcher in background
	watcher, err := fsnotify.NewWatcher()
	if err == nil {
		go watchFiles(watcher, p, root)
		defer watcher.Close()
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running sidegit: %v\n", err)
		os.Exit(1)
	}
}

func watchFiles(watcher *fsnotify.Watcher, p *tea.Program, root string) {
	// Watch root and immediate subdirectories that are git repos
	addWatchPaths(watcher, root)

	var (
		mu       sync.Mutex
		debounce *time.Timer
	)

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Ignore .git internal changes (except HEAD which indicates branch switch)
			if isGitInternalPath(event.Name) {
				continue
			}
			mu.Lock()
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(100*time.Millisecond, func() {
				p.Send(fileChangedMsg{})
			})
			mu.Unlock()

		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func addWatchPaths(watcher *fsnotify.Watcher, root string) {
	_ = watcher.Add(root)

	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		sub := filepath.Join(root, entry.Name())
		_ = watcher.Add(sub)

		// Watch one level deeper
		subEntries, err := os.ReadDir(sub)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() || subEntry.Name()[0] == '.' {
				continue
			}
			_ = watcher.Add(filepath.Join(sub, subEntry.Name()))
		}
	}
}

func isGitInternalPath(path string) bool {
	// Allow watching .git/HEAD for branch switches, but ignore most .git internals
	base := filepath.Base(path)
	dir := filepath.Dir(path)

	if filepath.Base(dir) == ".git" && base == "HEAD" {
		return false
	}

	// Walk up to check if any parent is .git
	for p := path; p != "/" && p != "."; p = filepath.Dir(p) {
		if filepath.Base(p) == ".git" {
			return true
		}
	}
	return false
}
