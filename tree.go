package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type NodeKind int

const (
	NodeRepo NodeKind = iota
	NodeDir
	NodeFile
)

type TreeNode struct {
	Kind      NodeKind
	Repo      *Repo
	File      *FileStatus
	DirPath   string // for NodeDir: the directory path
	RepoIndex int
	Depth     int  // indentation depth (0=repo, 1=dir/root file, 2=file under dir)
	Collapsed bool
	ParentDir int // index of parent dir node (-1 if none)
}

type TreeModel struct {
	nodes   []TreeNode
	visible []int
	cursor  int
	theme   Theme
}

func NewTreeModel(repos []Repo, theme Theme) TreeModel {
	var nodes []TreeNode
	for i := range repos {
		repoIdx := len(nodes)
		nodes = append(nodes, TreeNode{
			Kind:      NodeRepo,
			Repo:      &repos[i],
			RepoIndex: i,
			Depth:     0,
			ParentDir: -1,
		})

		// Group files by directory
		dirFiles := map[string][]*FileStatus{} // dir -> files
		for j := range repos[i].Files {
			f := &repos[i].Files[j]
			dir := filepath.Dir(f.Path)
			if dir == "." {
				dir = ""
			}
			dirFiles[dir] = append(dirFiles[dir], f)
		}

		// Collect all directory paths including intermediate ancestors
		dirSet := map[string]bool{}
		for dir := range dirFiles {
			if dir == "" {
				continue
			}
			parts := strings.Split(dir, "/")
			for k := 1; k <= len(parts); k++ {
				dirSet[strings.Join(parts[:k], "/")] = true
			}
		}
		var allDirs []string
		for d := range dirSet {
			allDirs = append(allDirs, d)
		}
		sort.Strings(allDirs)

		// Build directory nodes hierarchically
		dirNodeIdx := map[string]int{} // dir path -> node index
		var addDir func(dir string)
		addDir = func(dir string) {
			if _, exists := dirNodeIdx[dir]; exists {
				return
			}
			parts := strings.Split(dir, "/")
			depth := len(parts) // 1 for top-level, 2 for nested, etc.
			parentIdx := repoIdx
			if len(parts) > 1 {
				parentDir := strings.Join(parts[:len(parts)-1], "/")
				addDir(parentDir) // ensure parent exists
				parentIdx = dirNodeIdx[parentDir]
			}
			dirIdx := len(nodes)
			dirNodeIdx[dir] = dirIdx
			nodes = append(nodes, TreeNode{
				Kind:      NodeDir,
				DirPath:   parts[len(parts)-1], // show just the last segment
				Repo:      &repos[i],
				RepoIndex: i,
				Depth:     depth,
				ParentDir: parentIdx,
			})
			// Add files that belong directly to this directory
			if files, ok := dirFiles[dir]; ok {
				for _, f := range files {
					nodes = append(nodes, TreeNode{
						Kind:      NodeFile,
						File:      f,
						Repo:      &repos[i],
						RepoIndex: i,
						Depth:     depth + 1,
						ParentDir: dirIdx,
					})
				}
			}
		}
		for _, dir := range allDirs {
			addDir(dir)
		}

		// Then root-level files
		if rootFiles, ok := dirFiles[""]; ok {
			for _, f := range rootFiles {
				nodes = append(nodes, TreeNode{
					Kind:      NodeFile,
					File:      f,
					Repo:      &repos[i],
					RepoIndex: i,
					Depth:     1,
					ParentDir: repoIdx,
				})
			}
		}
	}

	tm := TreeModel{nodes: nodes, theme: theme}
	tm.rebuildVisible()
	return tm
}

func (tm *TreeModel) rebuildVisible() {
	tm.visible = nil
	for i, n := range tm.nodes {
		switch n.Kind {
		case NodeRepo:
			tm.visible = append(tm.visible, i)
		case NodeDir:
			// Visible if all ancestors are expanded
			if tm.isAncestorExpanded(n) {
				tm.visible = append(tm.visible, i)
			}
		case NodeFile:
			// Check all ancestors are expanded
			if tm.isAncestorExpanded(n) {
				tm.visible = append(tm.visible, i)
			}
		}
	}
	if tm.cursor >= len(tm.visible) {
		tm.cursor = max(0, len(tm.visible)-1)
	}
}

func (tm *TreeModel) isAncestorExpanded(n TreeNode) bool {
	if n.ParentDir < 0 {
		return true
	}
	parent := &tm.nodes[n.ParentDir]
	if parent.Collapsed {
		return false
	}
	return tm.isAncestorExpanded(*parent)
}

func (tm *TreeModel) MoveUp() {
	if tm.cursor > 0 {
		tm.cursor--
	}
}

func (tm *TreeModel) MoveDown() {
	if tm.cursor < len(tm.visible)-1 {
		tm.cursor++
	}
}

func (tm *TreeModel) ToggleCollapse() {
	node := tm.SelectedNode()
	if node == nil {
		return
	}
	if node.Kind == NodeRepo || node.Kind == NodeDir {
		node.Collapsed = !node.Collapsed
		tm.rebuildVisible()
	}
}

func (tm *TreeModel) SelectedNode() *TreeNode {
	if len(tm.visible) == 0 || tm.cursor < 0 || tm.cursor >= len(tm.visible) {
		return nil
	}
	return &tm.nodes[tm.visible[tm.cursor]]
}

func (tm *TreeModel) Len() int {
	return len(tm.visible)
}

func (tm *TreeModel) Render(width, height int) string {
	if len(tm.visible) == 0 {
		return lipgloss.NewStyle().
			Width(width).
			Height(height).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(lipgloss.Color(tm.theme.NoRepos)).
			Render("No git repositories found.\nRun sidegit in a directory containing git repos.")
	}

	startIdx := 0
	if tm.cursor >= height {
		startIdx = tm.cursor - height + 1
	}

	var lines []string
	cursorBg := lipgloss.Color(tm.theme.CursorBg)
	for i := startIdx; i < len(tm.visible) && len(lines) < height; i++ {
		node := tm.nodes[tm.visible[i]]
		selected := i == tm.cursor
		line := renderNode(node, selected, width, tm.theme, cursorBg)
		line = padRight(line, width, selected, cursorBg)
		lines = append(lines, line)
	}

	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}

	return strings.Join(lines, "\n")
}

func padRight(s string, width int, selected bool, cursorBg lipgloss.Color) string {
	visible := lipgloss.Width(s)
	pad := width - visible
	if pad <= 0 {
		return s
	}
	spaces := strings.Repeat(" ", pad)
	if selected {
		spaces = lipgloss.NewStyle().Background(cursorBg).Render(spaces)
	}
	return s + spaces
}

// truncateStr shortens a string from the right with "…" suffix.
func truncateStr(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	return s[:maxWidth-1] + "…"
}

// truncatePath shortens a path from the left to fit maxWidth, e.g. "…/Projects/gitbar"
func truncatePath(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}
	if maxWidth <= 3 {
		return "..."
	}

	parts := strings.Split(path, string(filepath.Separator))
	// Always keep the last segment (folder name)
	result := parts[len(parts)-1]
	if len(result)+2 > maxWidth {
		// Even the last segment is too long
		return "…" + result[len(result)-(maxWidth-1):]
	}

	// Add segments from the right until we'd exceed maxWidth
	for i := len(parts) - 2; i >= 0; i-- {
		candidate := parts[i] + string(filepath.Separator) + result
		if len(candidate)+1 > maxWidth { // +1 for the "…" prefix
			break
		}
		result = candidate
	}

	if result == path {
		return path
	}
	return "…" + string(filepath.Separator) + result
}

func renderNode(node TreeNode, selected bool, width int, theme Theme, cursorBg lipgloss.Color) string {
	var bg lipgloss.Style
	if selected {
		bg = lipgloss.NewStyle().Background(cursorBg)
	} else {
		bg = lipgloss.NewStyle()
	}

	indent := bg.Render(strings.Repeat("  ", node.Depth))
	sp := bg.Render(" ")

	switch node.Kind {
	case NodeRepo:
		arrow := "▾"
		if node.Collapsed {
			arrow = "▸"
		}
		branchStr := fmt.Sprintf("[%s]", node.Repo.Branch)
		countStr := fmt.Sprintf("(%d)", len(node.Repo.Files))

		// Available space after "▸ 📁 " (4 chars)
		avail := width - 4

		// Try to fit: name + " " + branch + " " + count
		nameSpace := avail - len(branchStr) - len(countStr) - 2
		if nameSpace >= 3 {
			// All three fit
			nameStr := truncatePath(node.Repo.RelPath, nameSpace)
			icon := bg.Foreground(lipgloss.Color(theme.FolderIcon)).Render("\uf07b")
			name := bg.Bold(true).Foreground(lipgloss.Color(theme.RepoName)).Render(nameStr)
			branch := bg.Bold(false).Foreground(lipgloss.Color(theme.BranchName)).Render(branchStr)
			fileCount := bg.Foreground(lipgloss.Color(theme.FileCount)).Render(countStr)
			arrowStyled := bg.Render(arrow)
			return arrowStyled + sp + icon + sp + name + sp + branch + sp + fileCount
		}

		// Drop count, try: name + " " + branch
		nameSpace = avail - len(branchStr) - 1
		if nameSpace >= 3 {
			nameStr := truncatePath(node.Repo.RelPath, nameSpace)
			icon := bg.Foreground(lipgloss.Color(theme.FolderIcon)).Render("\uf07b")
			name := bg.Bold(true).Foreground(lipgloss.Color(theme.RepoName)).Render(nameStr)
			branch := bg.Bold(false).Foreground(lipgloss.Color(theme.BranchName)).Render(branchStr)
			arrowStyled := bg.Render(arrow)
			return arrowStyled + sp + icon + sp + name + sp + branch
		}

		// Drop branch too, just: name (truncated)
		nameStr := truncatePath(node.Repo.RelPath, max(1, avail))
		icon := bg.Foreground(lipgloss.Color(theme.FolderIcon)).Render("\uf07b")
		name := bg.Bold(true).Foreground(lipgloss.Color(theme.RepoName)).Render(nameStr)
		arrowStyled := bg.Render(arrow)
		return arrowStyled + sp + icon + sp + name

	case NodeDir:
		arrow := "▾"
		if node.Collapsed {
			arrow = "▸"
		}
		// indent + arrow + sp + icon + sp + name
		fixedWidth := node.Depth*2 + 1 + 1 + 1 + 1
		dirName := truncateStr(node.DirPath, width-fixedWidth)
		icon := bg.Foreground(lipgloss.Color(theme.FolderIcon)).Render("\uf07b")
		name := bg.Bold(true).Foreground(lipgloss.Color(theme.DirName)).Render(dirName)
		arrowStyled := bg.Render(arrow)
		return indent + arrowStyled + sp + icon + sp + name

	case NodeFile:
		// indent + status + sp + icon + sp + name
		fixedWidth := node.Depth*2 + 1 + 1 + 1 + 1
		fileName := truncateStr(filepath.Base(node.File.Path), width-fixedWidth)
		styledStatus := styleStatus(node.File.Status, node.File.IsStaged, selected, theme, cursorBg)
		icon := fileIconStyled(node.File.Path, selected, theme, cursorBg)
		fileStyled := bg.Render(fileName)
		return indent + styledStatus + sp + icon + sp + fileStyled
	}
	return ""
}

func styleStatus(code StatusCode, staged bool, selected bool, theme Theme, cursorBg lipgloss.Color) string {
	s := string(code)
	base := lipgloss.NewStyle()
	if selected {
		base = base.Background(cursorBg)
	}
	if staged {
		return base.Foreground(lipgloss.Color(theme.StatusStaged)).Bold(true).Render(s)
	}
	switch code {
	case StatusAdded:
		return base.Foreground(lipgloss.Color(theme.StatusAdded)).Render(s)
	case StatusDeleted:
		return base.Foreground(lipgloss.Color(theme.StatusDeleted)).Render(s)
	case StatusModified:
		return base.Foreground(lipgloss.Color(theme.StatusModified)).Render(s)
	case StatusUntracked:
		return base.Foreground(lipgloss.Color(theme.StatusUntracked)).Render(s)
	default:
		return base.Render(s)
	}
}

// Nerd Font icon lookup by file extension (codepoints from nvim-web-devicons).
var nerdIcons = map[string]string{
	".go":         "\ue627",     // seti-go
	".js":         "\ue60c",     // seti-javascript
	".ts":         "\ue628",     // seti-typescript
	".tsx":        "\ue7ba",     // seti-react
	".jsx":        "\ue625",     // seti-react
	".py":         "\ue606",     // seti-python
	".rb":         "\ue791",     // seti-ruby
	".rs":         "\ue68b",     // seti-rust
	".java":       "\ue738",     // dev-java
	".c":          "\ue61e",     // seti-c
	".cpp":        "\ue61d",     // seti-cpp
	".h":          "\ue61e",     // seti-c
	".cs":         "\U000F0911", // md-language_csharp
	".php":        "\ue608",     // seti-php
	".swift":      "\ue755",     // dev-swift
	".kt":         "\ue634",     // seti-kotlin
	".html":       "\ue736",     // dev-html5
	".css":        "\ue6b8",     // seti-css
	".scss":       "\ue603",     // seti-sass
	".json":       "\ue60b",     // seti-json
	".yaml":       "\ue615",     // seti-yaml
	".yml":        "\ue615",     // seti-yaml
	".toml":       "\ue6b2",     // seti-toml
	".xml":        "\U000F05C0", // md-xml
	".md":         "\uf48a",     // oct-markdown
	".sh":         "\ue795",     // seti-shell
	".bash":       "\ue795",     // seti-shell
	".zsh":        "\ue795",     // seti-shell
	".fish":       "\ue795",     // seti-shell
	".sql":        "\ue706",     // dev-database
	".svg":        "\U000F0721", // md-svg
	".png":        "\ue60d",     // seti-image
	".jpg":        "\ue60d",     // seti-image
	".jpeg":       "\ue60d",     // seti-image
	".gif":        "\ue60d",     // seti-image
	".vue":        "\ue6a0",     // seti-vue
	".svelte":     "\ue697",     // seti-svelte
	".lua":        "\ue620",     // seti-lua
	".vim":        "\ue62b",     // seti-vim
	".lock":       "\ue672",     // seti-lock
	".env":        "\uf462",     // oct-key
	".gitignore":  "\ue702",     // dev-git
	".mod":        "\ue627",     // seti-go
	".sum":        "\ue627",     // seti-go
}

// Special full-name matches.
var nerdIconNames = map[string]string{
	"Dockerfile":  "\U000F01A8", // md-docker
	"Makefile":    "\ue673",     // seti-makefile
	"LICENSE":     "\ue62f",     // seti-license
	".gitignore":  "\ue702",     // dev-git
	"go.mod":      "\ue627",     // seti-go
	"go.sum":      "\ue627",     // seti-go
}

// Icon color map by extension.
var iconColors = map[string]string{
	".go": "#00ADD8", ".js": "#CBCB41", ".ts": "#519ABA", ".tsx": "#1354BF",
	".jsx": "#20C2E3", ".py": "#FFBC03", ".rb": "#701516", ".rs": "#DEA584",
	".java": "#CC3E44", ".html": "#E44D26", ".css": "#663399", ".scss": "#F55385",
	".json": "#CBCB41", ".yaml": "#6D8086", ".yml": "#6D8086", ".toml": "#9C4221",
	".md": "#DDDDDD", ".sh": "#4D5A5E", ".bash": "#4D5A5E", ".zsh": "#4D5A5E",
	".php": "#A074C4", ".swift": "#E37933", ".kt": "#7F52FF", ".lua": "#51A0CF",
	".vue": "#8DC149", ".svelte": "#FF3E00", ".sql": "#DAD8D8", ".lock": "#BBBBBB",
	".env": "#FAF743", ".gitignore": "#F54D27",
}

func fileIconStyled(path string, selected bool, theme Theme, cursorBg lipgloss.Color) string {
	name := filepath.Base(path)

	if icon, ok := nerdIconNames[name]; ok {
		return colorIcon(icon, name, selected, theme, cursorBg)
	}

	ext := strings.ToLower(filepath.Ext(name))
	if icon, ok := nerdIcons[ext]; ok {
		return colorIcon(icon, name, selected, theme, cursorBg)
	}

	base := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.DefaultIcon))
	if selected {
		base = base.Background(cursorBg)
	}
	return base.Render("\uf15b")
}

func colorIcon(icon, name string, selected bool, theme Theme, cursorBg lipgloss.Color) string {
	ext := strings.ToLower(filepath.Ext(name))
	base := lipgloss.NewStyle()
	if selected {
		base = base.Background(cursorBg)
	}
	if color, ok := iconColors[ext]; ok {
		return base.Foreground(lipgloss.Color(color)).Render(icon)
	}
	return base.Foreground(lipgloss.Color(theme.DefaultIcon)).Render(icon)
}
