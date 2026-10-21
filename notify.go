package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

type prChange struct {
	key  string // tells this change apart from others on the same PR
	text string
}

// prChanges lists the changes between two fetches of a PR that get a notification:
// the checks finished, or new review threads came in.
func prChanges(old, cur *PRStatus) []prChange {
	if old == nil || cur == nil || old.Number != cur.Number || cur.State != "OPEN" {
		return nil
	}
	var changes []prChange
	done := map[string]string{"SUCCESS": "checks passed", "FAILURE": "checks failed", "ERROR": "checks failed"}
	// A new push restarts the checks, which can finish between two fetches.
	if text, ok := done[cur.Checks]; ok && (done[old.Checks] == "" || old.Head != cur.Head) {
		changes = append(changes, prChange{"checks:" + cur.Checks, text})
	}
	if n := cur.Unresolved - old.Unresolved; n > 0 {
		text := fmt.Sprintf("%d new review threads", n)
		if n == 1 {
			text = "1 new review thread"
		}
		changes = append(changes, prChange{fmt.Sprintf("threads:%d", cur.Unresolved), text})
	}
	return changes
}

// notifyPRChangesCmd posts a macOS notification through iTerm2 (OSC 9) for each PR change.
func notifyPRChangesCmd(old, cur prsFetchedMsg) tea.Cmd {
	return func() tea.Msg {
		for k, pr := range cur {
			for _, c := range prChanges(old[k], pr) {
				if claimNotification(fmt.Sprintf("%s#%d@%s:%s", k.path, pr.Number, pr.Head, c.key)) {
					fmt.Fprintf(os.Stdout, "\x1b]9;%s #%d: %s\x07", filepath.Base(k.path), pr.Number, c.text)
				}
			}
		}
		return nil
	}
}

// claimNotification is true for the first sidegit pane that sees a change, so a
// repo open in several panes gives one notification.
func claimNotification(key string) bool {
	dir := filepath.Join(cacheDir(), "notified")
	if os.MkdirAll(dir, 0o755) != nil {
		return false
	}
	sum := sha1.Sum([]byte(key))
	f, err := os.OpenFile(filepath.Join(dir, hex.EncodeToString(sum[:])), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
