package main

import (
	"os"
	"path/filepath"
	"sort"
)

type Repo struct {
	Path    string
	RelPath string
	Branch  string
	Files   []FileStatus
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

	return repos, nil
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
	files, _ := GetStatus(repoPath)

	return Repo{
		Path:    repoPath,
		RelPath: rel,
		Branch:  branch,
		Files:   files,
	}
}
