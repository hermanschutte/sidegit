package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ClaudeEdit is the Claude Code session that last edited a file.
type ClaudeEdit struct {
	Session string    `json:"session"`
	Title   string    `json:"title"`
	Time    time.Time `json:"time"`
}

// cacheDir holds the state that all sidegit panes share.
func cacheDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "sidegit")
}

// editRecordPath is the record of one edited file, named by the hash of its path.
func editRecordPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	sum := sha1.Sum([]byte(path))
	return filepath.Join(cacheDir(), "edits", hex.EncodeToString(sum[:]))
}

// runClaudeHook records which session edited a file. Claude Code runs it as a
// PostToolUse hook for Edit, Write, MultiEdit and NotebookEdit, with the event JSON on stdin.
func runClaudeHook() error {
	var event struct {
		SessionID      string `json:"session_id"`
		TranscriptPath string `json:"transcript_path"`
		ToolInput      struct {
			FilePath     string `json:"file_path"`
			NotebookPath string `json:"notebook_path"`
		} `json:"tool_input"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&event); err != nil {
		return err
	}
	path := event.ToolInput.FilePath
	if path == "" {
		path = event.ToolInput.NotebookPath
	}
	if path == "" || event.SessionID == "" {
		return nil
	}
	data, err := json.Marshal(ClaudeEdit{Session: event.SessionID, Title: sessionTitle(event.TranscriptPath), Time: time.Now()})
	if err != nil {
		return err
	}
	record := editRecordPath(path)
	if err := os.MkdirAll(filepath.Dir(record), 0o755); err != nil {
		return err
	}
	return os.WriteFile(record, data, 0o644)
}

// lastEdit returns the session that last edited the file. It returns nil when the
// file changed again after that edit, for example by hand or by git.
func lastEdit(path string, modTime time.Time) *ClaudeEdit {
	data, err := os.ReadFile(editRecordPath(path))
	if err != nil {
		return nil
	}
	var e ClaudeEdit
	// The record is written just after the edit; a formatter hook can touch the file a little later.
	if json.Unmarshal(data, &e) != nil || modTime.After(e.Time.Add(10*time.Second)) {
		return nil
	}
	return &e
}

// sessionTitle finds the session name that the terminal tab shows: the name set
// with /rename, or else the title that Claude Code made. Both lines repeat through
// the transcript, so the end of the file is enough, and a transcript can be hundreds of MB.
func sessionTitle(transcript string) string {
	f, err := os.Open(transcript)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	for size := int64(1 << 20); ; size *= 4 {
		start := max(0, info.Size()-size)
		buf := make([]byte, info.Size()-start)
		if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
			return ""
		}
		if title := lastTitle(buf); title != "" || start == 0 || size >= 64<<20 {
			return title
		}
	}
}

func lastTitle(buf []byte) string {
	var custom, ai string
	for _, line := range bytes.Split(buf, []byte("\n")) {
		if !bytes.Contains(line, []byte(`"type":"custom-title"`)) && !bytes.Contains(line, []byte(`"type":"ai-title"`)) {
			continue
		}
		var entry struct {
			CustomTitle string `json:"customTitle"`
			AiTitle     string `json:"aiTitle"`
		}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		if entry.CustomTitle != "" {
			custom = entry.CustomTitle
		}
		if entry.AiTitle != "" {
			ai = entry.AiTitle
		}
	}
	if custom != "" {
		return custom
	}
	return ai
}
