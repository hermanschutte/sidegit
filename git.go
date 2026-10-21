package main

import (
	"bytes"
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
	Added    int // lines added compared to HEAD (all lines for an untracked file)
	Deleted  int // lines deleted compared to HEAD
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

type GitStatus struct {
	Files  []FileStatus
	Ahead  int
	Behind int
}

func GetStatus(repoPath string) (GitStatus, error) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain=v2", "--branch", "--untracked-files=all")
	out, err := cmd.Output()
	if err != nil {
		return GitStatus{}, fmt.Errorf("git status failed: %w", err)
	}

	var result GitStatus
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "# branch.ab ") {
			fmt.Sscanf(line, "# branch.ab +%d -%d", &result.Ahead, &result.Behind)
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 ") {
			fs := parseOrdinaryEntry(line)
			if fs != nil {
				result.Files = append(result.Files, *fs)
			}
		} else if strings.HasPrefix(line, "? ") {
			path := line[2:]
			result.Files = append(result.Files, FileStatus{
				Path:   path,
				Status: StatusUntracked,
			})
		}
	}

	counts := lineCounts(repoPath)
	for i := range result.Files {
		f := &result.Files[i]
		if f.Status == StatusUntracked {
			f.Added = countLines(filepath.Join(repoPath, f.Path))
			continue
		}
		if c, ok := counts[f.Path]; ok {
			f.Added, f.Deleted = c[0], c[1]
		}
	}

	return result, nil
}

// lineCounts returns added and deleted lines per tracked file, staged and unstaged
// together. Binary files and repos without commits get no entry.
func lineCounts(repoPath string) map[string][2]int {
	counts := map[string][2]int{}
	cmd := exec.Command("git", "-C", repoPath, "diff", "HEAD", "--numstat", "--no-renames", "-z")
	out, err := cmd.Output()
	if err != nil {
		return counts
	}
	// Each record is "added\tdeleted\tpath\0"; binary files have "-" for both numbers.
	for _, rec := range strings.Split(string(out), "\x00") {
		parts := strings.SplitN(rec, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		var added, deleted int
		if _, err := fmt.Sscanf(parts[0]+" "+parts[1], "%d %d", &added, &deleted); err != nil {
			continue
		}
		counts[parts[2]] = [2]int{added, deleted}
	}
	return counts
}

// countLines counts the lines of an untracked text file. Folders, binary files
// and files over 1 MB count as 0.
func countLines(path string) int {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > 1<<20 {
		return 0
	}
	data, err := os.ReadFile(path)
	if err != nil || bytes.IndexByte(data, 0) >= 0 {
		return 0
	}
	n := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n
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

func DiscardAllChanges(repoPath, filePath string, isUntracked bool) error {
	if isUntracked {
		return os.Remove(filepath.Join(repoPath, filePath))
	}
	// Unstage first (ignore error — file may not be staged)
	unstage := exec.Command("git", "-C", repoPath, "reset", "HEAD", "--", filePath)
	_ = unstage.Run()
	// Discard working tree changes
	cmd := exec.Command("git", "-C", repoPath, "checkout", "--", filePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %s", out)
	}
	return nil
}

func ListBranches(repoPath string) ([]string, string, error) {
	current := FindBranch(repoPath)
	cmd := exec.Command("git", "-C", repoPath, "branch", "--format=%(refname:short)")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, current, fmt.Errorf("git branch: %s", out)
	}
	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, current, nil
}

type Worktree struct {
	Path   string
	Branch string // short branch name, empty when detached
}

// ListWorktrees returns all worktrees of the repo. The first entry is the main worktree.
func ListWorktrees(repoPath string) ([]Worktree, error) {
	cmd := exec.Command("git", "-C", repoPath, "worktree", "list", "--porcelain")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %s", out)
	}
	var worktrees []Worktree
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			worktrees = append(worktrees, Worktree{Path: strings.TrimPrefix(line, "worktree ")})
		} else if strings.HasPrefix(line, "branch ") && len(worktrees) > 0 {
			worktrees[len(worktrees)-1].Branch = strings.TrimPrefix(line, "branch refs/heads/")
		}
	}
	return worktrees, nil
}

// RemoveWorktree deletes a linked worktree folder. The branch stays.
// force also removes a worktree that has uncommitted changes.
func RemoveWorktree(mainRepoPath, worktreePath string, force bool) error {
	args := []string{"-C", mainRepoPath, "worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git worktree remove: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func CheckoutBranch(repoPath, branch string) error {
	cmd := exec.Command("git", "-C", repoPath, "checkout", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout: %s", out)
	}
	return nil
}

func GitPull(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "pull")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git pull: %s", out)
	}
	return nil
}

func GitPush(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "push")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %s", out)
	}
	return nil
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
