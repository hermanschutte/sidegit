package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Repo struct {
	Path    string
	RelPath string
	Branch  string
	Files   []FileStatus
	Ahead   int
	Behind  int
	// Worktree is true for a linked worktree. It follows its main repo in the list.
	Worktree bool
}

func ScanRepos(root string) ([]Repo, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	var repos []Repo

	// Check if root itself is a git repo
	if isGitRepo(root) {
		repos = append(repos, buildRepo(root, root))
	}

	// Scan immediate subdirectories
	entries, err := os.ReadDir(root)
	if err != nil {
		return repos, nil // return what we have
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		sub := filepath.Join(root, entry.Name())
		if isGitRepo(sub) {
			repos = append(repos, buildRepo(root, sub))
		}
		// Also check one level deeper
		subEntries, err := os.ReadDir(sub)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() || subEntry.Name()[0] == '.' {
				continue
			}
			deep := filepath.Join(sub, subEntry.Name())
			if isGitRepo(deep) {
				repos = append(repos, buildRepo(root, deep))
			}
		}
	}

	// Sort by relative path, but keep root (".") first
	sort.Slice(repos, func(i, j int) bool {
		if repos[i].RelPath == "." {
			return true
		}
		if repos[j].RelPath == "." {
			return false
		}
		return repos[i].RelPath < repos[j].RelPath
	})

	// Put each repo's linked worktrees directly after it
	var withWorktrees []Repo
	for _, repo := range repos {
		worktrees, err := ListWorktrees(repo.Path)
		if err != nil || len(worktrees) < 2 {
			withWorktrees = append(withWorktrees, repo)
			continue
		}
		// A worktree inside the repo shows as an untracked folder. Its own row shows it.
		var files []FileStatus
		for _, f := range repo.Files {
			abs := filepath.Join(repo.Path, strings.TrimSuffix(f.Path, "/"))
			if !isWorktreePath(abs, worktrees[1:]) {
				files = append(files, f)
			}
		}
		repo.Files = files
		withWorktrees = append(withWorktrees, repo)
		for _, wt := range worktrees[1:] {
			if _, err := os.Stat(wt.Path); err != nil {
				continue // worktree folder was removed
			}
			w := buildRepo(root, wt.Path)
			w.RelPath = filepath.Base(wt.Path)
			w.Worktree = true
			withWorktrees = append(withWorktrees, w)
		}
	}

	return withWorktrees, nil
}

func isWorktreePath(path string, worktrees []Worktree) bool {
	for _, wt := range worktrees {
		if path == wt.Path || strings.HasPrefix(path, wt.Path+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func isGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

func buildRepo(root, repoPath string) Repo {
	rel, err := filepath.Rel(root, repoPath)
	if err != nil {
		rel = repoPath
	}
	if rel == "" || rel == "." {
		// Use the absolute path for the root repo
		rel = repoPath
	}

	branch := FindBranch(repoPath)
	status, _ := GetStatus(repoPath)

	return Repo{
		Path:    repoPath,
		RelPath: rel,
		Branch:  branch,
		Files:   status.Files,
		Ahead:   status.Ahead,
		Behind:  status.Behind,
	}
}
