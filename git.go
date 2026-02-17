package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type StatusCode string

const (
	StatusModified  StatusCode = "M"
	StatusAdded     StatusCode = "A"
	StatusDeleted   StatusCode = "D"
	StatusRenamed   StatusCode = "R"
	StatusCopied    StatusCode = "C"
	StatusUntracked StatusCode = "?"
)

type FileStatus struct {
	Path     string
	Status   StatusCode
	IsStaged bool
}

func FindBranch(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}

	// Fallback for repos with no commits: read .git/HEAD directly
	data, err := os.ReadFile(filepath.Join(repoPath, ".git", "HEAD"))
	if err != nil {
		return "unknown"
	}
	ref := strings.TrimSpace(string(data))
	if strings.HasPrefix(ref, "ref: refs/heads/") {
		return strings.TrimPrefix(ref, "ref: refs/heads/")
	}
	return ref
}

func GetStatus(repoPath string) ([]FileStatus, error) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain=v2", "--branch", "--untracked-files=all")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	var files []FileStatus
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			fs := parseOrdinaryEntry(line)
			if fs != nil {
				files = append(files, *fs)
			}
		} else if strings.HasPrefix(line, "? ") {
			path := line[2:]
			files = append(files, FileStatus{
				Path:   path,
				Status: StatusUntracked,
			})
		}
	}

	return files, nil
}

func parseOrdinaryEntry(line string) *FileStatus {
	// Format: 1 XY sub mH mI mW hH hI path
	// or:     2 XY sub mH mI mW hH hI X### origPath\tpath
	fields := strings.Fields(line)
	if len(fields) < 9 {
		return nil
	}

	xy := fields[1]
	stagedCode := xy[0]
	unstagedCode := xy[1]

	var path string
	if fields[0] == "2" {
		// Rename entry — path is in the last field after tab
		lastField := fields[len(fields)-1]
		parts := strings.SplitN(lastField, "\t", 2)
		if len(parts) == 2 {
			path = parts[1]
		} else {
			path = lastField
		}
	} else {
		path = fields[len(fields)-1]
	}

	// Prefer showing unstaged changes; if none, show staged
	if unstagedCode != '.' {
		return &FileStatus{
			Path:   path,
			Status: mapStatusByte(unstagedCode),
		}
	}
	if stagedCode != '.' {
		return &FileStatus{
			Path:     path,
			Status:   mapStatusByte(stagedCode),
			IsStaged: true,
		}
	}

	return nil
}

func mapStatusByte(b byte) StatusCode {
	switch b {
	case 'M':
		return StatusModified
	case 'A':
		return StatusAdded
	case 'D':
		return StatusDeleted
	case 'R':
		return StatusRenamed
	case 'C':
		return StatusCopied
	default:
		return StatusModified
	}
}

func GetDiff(repoPath, filePath string) (string, error) {
	absFile := filepath.Join(repoPath, filePath)

	// Check if the file is untracked
	cmd := exec.Command("git", "-C", repoPath, "ls-files", "--error-unmatch", filePath)
	if err := cmd.Run(); err != nil {
		// Untracked file — diff against /dev/null
		cmd = exec.Command("git", "-C", repoPath, "diff", "--no-index", "--color=always", "--", "/dev/null", absFile)
		out, _ := cmd.Output()
		if len(out) == 0 {
			return "(new untracked file)", nil
		}
		return string(out), nil
	}

	// Tracked file — normal diff
	cmd = exec.Command("git", "-C", repoPath, "diff", "--color=always", "--", filePath)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	if len(out) == 0 {
		// Maybe staged — try diff --cached
		cmd = exec.Command("git", "-C", repoPath, "diff", "--cached", "--color=always", "--", filePath)
		out, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git diff --cached failed: %w", err)
		}
		if len(out) == 0 {
			return "(no changes)", nil
		}
	}
	return string(out), nil
}
