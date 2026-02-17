package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Theme struct {
	CursorBg        string `yaml:"cursor_bg"`
	BorderFocused   string `yaml:"border_focused"`
	BorderNormal    string `yaml:"border_normal"`
	Title           string `yaml:"title"`
	StatusBar       string `yaml:"status_bar"`
	NoRepos         string `yaml:"no_repos"`
	RepoName        string `yaml:"repo_name"`
	BranchName      string `yaml:"branch_name"`
	FileCount       string `yaml:"file_count"`
	FolderIcon      string `yaml:"folder_icon"`
	DirName         string `yaml:"dir_name"`
	StatusStaged    string `yaml:"status_staged"`
	StatusAdded     string `yaml:"status_added"`
	StatusDeleted   string `yaml:"status_deleted"`
	StatusModified  string `yaml:"status_modified"`
	StatusUntracked string `yaml:"status_untracked"`
	DefaultIcon     string `yaml:"default_icon"`
}

func DefaultTheme() Theme {
	return Theme{
		CursorBg:        "237",
		BorderFocused:   "12",
		BorderNormal:    "8",
		Title:           "14",
		StatusBar:       "8",
		NoRepos:         "8",
		RepoName:        "12",
		BranchName:      "13",
		FileCount:       "7",
		FolderIcon:      "7",
		DirName:         "7",
		StatusStaged:    "10",
		StatusAdded:     "10",
		StatusDeleted:   "9",
		StatusModified:  "11",
		StatusUntracked: "8",
		DefaultIcon:     "7",
	}
}

type Config struct {
	DiffPosition string `yaml:"diff_position"`
	ScanDepth    int    `yaml:"scan_depth"`
	Theme        Theme  `yaml:"theme"`
}

func DefaultConfig() Config {
	return Config{
		DiffPosition: "right",
		ScanDepth:    1,
		Theme:        DefaultTheme(),
	}
}

func applyThemeDefaults(t *Theme) {
	d := DefaultTheme()
	if t.CursorBg == "" {
		t.CursorBg = d.CursorBg
	}
	if t.BorderFocused == "" {
		t.BorderFocused = d.BorderFocused
	}
	if t.BorderNormal == "" {
		t.BorderNormal = d.BorderNormal
	}
	if t.Title == "" {
		t.Title = d.Title
	}
	if t.StatusBar == "" {
		t.StatusBar = d.StatusBar
	}
	if t.NoRepos == "" {
		t.NoRepos = d.NoRepos
	}
	if t.RepoName == "" {
		t.RepoName = d.RepoName
	}
	if t.BranchName == "" {
		t.BranchName = d.BranchName
	}
	if t.FileCount == "" {
		t.FileCount = d.FileCount
	}
	if t.FolderIcon == "" {
		t.FolderIcon = d.FolderIcon
	}
	if t.DirName == "" {
		t.DirName = d.DirName
	}
	if t.StatusStaged == "" {
		t.StatusStaged = d.StatusStaged
	}
	if t.StatusAdded == "" {
		t.StatusAdded = d.StatusAdded
	}
	if t.StatusDeleted == "" {
		t.StatusDeleted = d.StatusDeleted
	}
	if t.StatusModified == "" {
		t.StatusModified = d.StatusModified
	}
	if t.StatusUntracked == "" {
		t.StatusUntracked = d.StatusUntracked
	}
	if t.DefaultIcon == "" {
		t.DefaultIcon = d.DefaultIcon
	}
}

func LoadConfig() Config {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	configDir := filepath.Join(home, ".config", "sidegit")
	configFile := filepath.Join(configDir, "config.yaml")

	data, err := os.ReadFile(configFile)
	if err != nil {
		// Create default config file
		_ = os.MkdirAll(configDir, 0755)
		defaultData, _ := yaml.Marshal(cfg)
		_ = os.WriteFile(configFile, defaultData, 0644)
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)
	applyThemeDefaults(&cfg.Theme)

	// Validate
	if cfg.DiffPosition != "right" && cfg.DiffPosition != "bottom" {
		cfg.DiffPosition = "right"
	}
	if cfg.ScanDepth < 1 {
		cfg.ScanDepth = 1
	}

	return cfg
}
