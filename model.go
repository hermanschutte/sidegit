package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type panel int

const (
	panelTree panel = iota
	panelDiff
)

// Messages
type reposScannedMsg struct {
	repos []Repo
}

type diffLoadedMsg struct {
	content string
	file    string
}

type fileChangedMsg struct{}
type pollTickMsg time.Time

type menuOption struct {
	key    string         // shortcut key displayed (e.g. "x", "u"), empty for Cancel
	label  string         // display text
	action func() tea.Cmd // nil means cancel/close
}

// Model
type model struct {
	repos        []Repo
	tree         TreeModel
	diffOpen     bool
	diffContent  string
	diffFile     string
	diffViewport viewport.Model
	config       Config
	width        int
	height       int
	focused      panel
	ready        bool
	scanRoot     string

	menuOpen    bool
	menuTitle   string
	menuOptions []menuOption
	menuCursor  int

}

func initialModel(cfg Config, root string) model {
	return model{
		config:   cfg,
		scanRoot: root,
	}
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{scanReposCmd(m.scanRoot)}
	if m.config.PollInterval > 0 {
		cmds = append(cmds, pollTickCmd(m.config.PollInterval))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.diffViewport = viewport.New(m.diffWidth(), m.diffHeight())
		m.diffViewport.SetContent(m.diffContent)
		return m, nil

	case reposScannedMsg:
		m.repos = msg.repos
		m.tree = NewTreeModel(m.repos, m.config.Theme)
		return m, nil

	case diffLoadedMsg:
		m.diffContent = msg.content
		m.diffFile = msg.file
		m.diffOpen = true
		m.diffViewport = viewport.New(m.diffWidth(), m.diffHeight())
		m.diffViewport.SetContent(m.diffContent)
		return m, nil

	case fileChangedMsg:
		return m, scanReposCmd(m.scanRoot)

	case editorFinishedMsg:
		return m, scanReposCmd(m.scanRoot)

	case pollTickMsg:
		cmds := []tea.Cmd{scanReposCmd(m.scanRoot)}
		if m.config.PollInterval > 0 {
			cmds = append(cmds, pollTickCmd(m.config.PollInterval))
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Update viewport if focused on diff
	if m.focused == panelDiff && m.diffOpen {
		var cmd tea.Cmd
		m.diffViewport, cmd = m.diffViewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *model) closeMenu() {
	m.menuOpen = false
	m.menuTitle = ""
	m.menuOptions = nil
	m.menuCursor = 0
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Intercept keys when menu is open
	if m.menuOpen {
		switch msg.String() {
		case "up", "k":
			if m.menuCursor > 0 {
				m.menuCursor--
			}
		case "down", "j":
			if m.menuCursor < len(m.menuOptions)-1 {
				m.menuCursor++
			}
		case "enter":
			opt := m.menuOptions[m.menuCursor]
			m.closeMenu()
			if opt.action != nil {
				return m, opt.action()
			}
		case "esc":
			m.closeMenu()
		default:
			// Check shortcut keys
			key := msg.String()
			for _, opt := range m.menuOptions {
				if opt.key == key {
					m.closeMenu()
					if opt.action != nil {
						return m, opt.action()
					}
					return m, nil
				}
			}
		}
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		if m.focused == panelTree {
			m.tree.MoveUp()
		} else {
			var cmd tea.Cmd
			m.diffViewport, cmd = m.diffViewport.Update(msg)
			return m, cmd
		}

	case "down", "j":
		if m.focused == panelTree {
			m.tree.MoveDown()
		} else {
			var cmd tea.Cmd
			m.diffViewport, cmd = m.diffViewport.Update(msg)
			return m, cmd
		}

	case "enter":
		if m.focused == panelTree {
			node := m.tree.SelectedNode()
			if node != nil && node.Kind == NodeFile {
				return m, loadDiffCmd(node.Repo.Path, node.File.Path)
			}
		}

	case "esc":
		m.diffOpen = false
		m.focused = panelTree

	case "tab":
		if m.diffOpen {
			if m.focused == panelTree {
				m.focused = panelDiff
			} else {
				m.focused = panelTree
			}
		}

	case "c", "e":
		if m.focused == panelTree {
			m.tree.ToggleCollapse()
		}

	case "o":
		if m.focused == panelTree {
			node := m.tree.SelectedNode()
			if node != nil && node.Kind == NodeFile {
				return m, openInEditorCmd(node.Repo.Path, node.File.Path)
			}
		}

	case "d":
		if m.focused == panelTree {
			node := m.tree.SelectedNode()
			if node != nil && node.Kind == NodeFile {
				repoPath := node.Repo.Path
				filePath := node.File.Path
				isUntracked := node.File.Status == StatusUntracked
				discardAll := func() tea.Cmd {
					return func() tea.Msg {
						_ = DiscardAllChanges(repoPath, filePath, isUntracked)
						return fileChangedMsg{}
					}
				}
				m.menuTitle = "Discard changes"
				m.menuOptions = []menuOption{
					{key: "x", label: "Discard all changes", action: discardAll},
					{label: "Cancel"},
				}
				m.menuCursor = 0
				m.menuOpen = true
			}
		}

	case "p":
		if m.config.DiffPosition == "right" {
			m.config.DiffPosition = "bottom"
		} else {
			m.config.DiffPosition = "right"
		}
		if m.diffOpen {
			m.diffViewport = viewport.New(m.diffWidth(), m.diffHeight())
			m.diffViewport.SetContent(m.diffContent)
		}

	case "r":
		return m, scanReposCmd(m.scanRoot)
	}

	return m, nil
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	statusBar := m.renderStatusBar()
	// 1 row status bar + 1 row margin bottom
	contentHeight := m.height - 2
	// 2 columns margin (1 left + 1 right)
	contentWidth := m.width - 2

	var content string
	if !m.diffOpen {
		content = m.renderTreePanel(contentWidth, contentHeight)
	} else {
		content = m.renderSplitView(contentWidth, contentHeight)
	}

	outer := lipgloss.NewStyle().
		MarginLeft(1).
		Render(content)

	statusBarWithMargin := lipgloss.NewStyle().MarginBottom(1).MarginLeft(1).Render(statusBar)

	view := lipgloss.JoinVertical(lipgloss.Left, outer, statusBarWithMargin)

	if m.menuOpen {
		view = m.renderMenu()
	}

	return view
}

func (m model) renderTreePanel(width, height int) string {
	borderColor := m.config.Theme.BorderNormal
	if m.focused == panelTree {
		borderColor = m.config.Theme.BorderFocused
	}

	return renderBorderedPanel("Files", m.tree.Render(width-2, height-2), width, height, borderColor, m.config.Theme.Title)
}

func (m model) renderSplitView(width, height int) string {
	if m.config.DiffPosition == "bottom" {
		treeH := height / 2
		diffH := height - treeH
		tree := m.renderTreePanel(width, treeH)
		diff := m.renderDiffPanel(width, diffH)
		return lipgloss.JoinVertical(lipgloss.Left, tree, diff)
	}

	// Right (default)
	treeW := width * 2 / 5
	diffW := width - treeW
	tree := m.renderTreePanel(treeW, height)
	diff := m.renderDiffPanel(diffW, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, tree, diff)
}

func (m model) renderDiffPanel(width, height int) string {
	borderColor := m.config.Theme.BorderNormal
	if m.focused == panelDiff {
		borderColor = m.config.Theme.BorderFocused
	}

	innerWidth := width - 2
	innerHeight := height - 2

	m.diffViewport.Width = innerWidth
	m.diffViewport.Height = innerHeight

	return renderBorderedPanel("Diff: "+m.diffFile, m.diffViewport.View(), width, height, borderColor, m.config.Theme.Title)
}

// renderBorderedPanel draws a box with a title embedded in the top border.
func renderBorderedPanel(title, content string, width, height int, borderColor, titleColor string) string {
	bc := lipgloss.Color(borderColor)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(titleColor))

	border := lipgloss.NormalBorder()
	innerWidth := width - 2 // left + right border chars

	// Top border with title: ┌─ Title ─────────┐
	// Truncate title if it would overflow (keep room for "┌─ " + " ─┐")
	maxTitle := innerWidth - 5
	if maxTitle > 0 && len(title) > maxTitle {
		title = title[:maxTitle-1] + "…"
	}
	titleStr := titleStyle.Render(title)
	titleVisible := lipgloss.Width(titleStr)
	lineAfter := innerWidth - titleVisible - 3
	if lineAfter < 0 {
		lineAfter = 0
	}
	topLine := lipgloss.NewStyle().Foreground(bc).Render(
		string(border.TopLeft)+string(border.Top)+" ") +
		titleStr +
		lipgloss.NewStyle().Foreground(bc).Render(
			" "+strings.Repeat(string(border.Top), lineAfter)+string(border.TopRight))

	// Content with side borders — explicitly clip to exact height
	innerHeight := height - 2 // minus top and bottom border rows
	contentLines := strings.Split(content, "\n")

	// Clip or pad to exact innerHeight
	for len(contentLines) < innerHeight {
		contentLines = append(contentLines, strings.Repeat(" ", innerWidth))
	}
	contentLines = contentLines[:innerHeight]

	borderStyle := lipgloss.NewStyle().Foreground(bc)
	left := borderStyle.Render(string(border.Left))
	right := borderStyle.Render(string(border.Right))
	emptyRight := strings.Repeat(" ", innerWidth)

	var rows []string
	rows = append(rows, topLine)
	for _, line := range contentLines {
		// Pad line to innerWidth if needed
		vis := lipgloss.Width(line)
		if vis < innerWidth {
			line = line + strings.Repeat(" ", innerWidth-vis)
		}
		_ = emptyRight
		rows = append(rows, left+line+right)
	}

	// Bottom border
	bottomLine := borderStyle.Render(
		string(border.BottomLeft) + strings.Repeat(string(border.Bottom), innerWidth) + string(border.BottomRight))
	rows = append(rows, bottomLine)

	return strings.Join(rows, "\n")
}

func (m model) renderMenu() string {
	cursorBg := lipgloss.Color(m.config.Theme.CursorBg)
	borderColor := m.config.Theme.BorderFocused

	// Full window width minus outer margin (1 left + 1 right)
	boxWidth := m.width - 2
	innerWidth := boxWidth - 2 // border chars

	// Build option lines
	var lines []string
	for i, opt := range m.menuOptions {
		selected := i == m.menuCursor
		bg := lipgloss.NewStyle()
		if selected {
			bg = bg.Background(cursorBg)
		}

		var line string
		if opt.key != "" {
			keyStyled := bg.Foreground(lipgloss.Color(m.config.Theme.Title)).Render(opt.key)
			labelStyled := bg.Foreground(lipgloss.NoColor{}).Render(" " + opt.label)
			line = keyStyled + labelStyled
		} else {
			line = bg.Render("  " + opt.label)
		}

		// Pad to full inner width with the same background
		vis := lipgloss.Width(line)
		if vis < innerWidth {
			line = line + bg.Render(strings.Repeat(" ", innerWidth-vis))
		}

		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	boxHeight := len(m.menuOptions) + 2

	box := renderBorderedPanel(m.menuTitle, content, boxWidth, boxHeight, borderColor, m.config.Theme.Title)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
}

func (m model) renderStatusBar() string {
	totalFiles := 0
	for _, r := range m.repos {
		totalFiles += len(r.Files)
	}

	left := fmt.Sprintf(" %d repo(s) | %d file(s)", len(m.repos), totalFiles)
	hints := "(q) quit  (↵) diff  (esc) close  (⇥) switch  (c) fold  (o) open  (d) discard  (p) layout  (r) refresh"

	full := left + " | " + strings.Repeat(" ", max(1, m.width-len(left)-len(hints)-3)) + hints

	// Truncate to fit in one line
	if len(full) > m.width {
		if m.width > 3 {
			full = full[:m.width-3] + "..."
		} else {
			full = full[:m.width]
		}
	}

	return lipgloss.NewStyle().
		Width(m.width).
		MaxHeight(1).
		Foreground(lipgloss.Color(m.config.Theme.StatusBar)).
		Render(full)
}

func (m model) diffWidth() int {
	contentWidth := m.width - 2
	if m.config.DiffPosition == "bottom" {
		return contentWidth - 2
	}
	return (contentWidth - contentWidth*2/5) - 2
}

func (m model) diffHeight() int {
	contentHeight := m.height - 2
	if m.config.DiffPosition == "bottom" {
		return contentHeight/2 - 2
	}
	return contentHeight - 2
}

// Commands
func scanReposCmd(root string) tea.Cmd {
	return func() tea.Msg {
		repos, _ := ScanRepos(root)
		return reposScannedMsg{repos: repos}
	}
}

func loadDiffCmd(repoPath, filePath string) tea.Cmd {
	return func() tea.Msg {
		content, err := GetDiff(repoPath, filePath)
		if err != nil {
			content = fmt.Sprintf("Error loading diff: %v", err)
		}
		return diffLoadedMsg{content: content, file: filePath}
	}
}

func pollTickCmd(seconds int) tea.Cmd {
	return tea.Tick(time.Duration(seconds)*time.Second, func(t time.Time) tea.Msg {
		return pollTickMsg(t)
	})
}

type editorFinishedMsg struct{ err error }

func openInEditorCmd(repoPath, filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	absPath := filepath.Join(repoPath, filePath)
	parts := strings.Fields(editor)
	args := append(parts[1:], absPath)
	c := exec.Command(parts[0], args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorFinishedMsg{err: err}
	})
}
