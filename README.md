# sidegit

A terminal UI for monitoring git changes across multiple repositories. Runs in the directory where your projects live and shows a unified tree of every repo with uncommitted changes.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Install

```
go install github.com/hermanschutte/sidegit@latest
```

Or build from source:

```
git clone https://github.com/hermanschutte/sidegit.git
cd sidegit
go build -o sidegit .
```

## Usage

Run `sidegit` in a directory containing git repositories:

```
cd ~/Projects
sidegit
```

It scans the current directory and up to two levels deep for git repos with uncommitted changes.

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` / `↑` / `↓` | Navigate |
| `Enter` | Show diff for selected file |
| `Tab` | Switch between tree and diff panels |
| `Esc` | Close diff panel |
| `c` / `e` | Collapse/expand repo or directory |
| `o` | Open file in `$EDITOR` |
| `d` | Discard changes (opens confirmation menu) |
| `p` | Toggle diff panel position (right/bottom) |
| `r` | Refresh |
| `q` | Quit |

## Configuration

Config lives at `~/.config/sidegit/config.yaml`. A default file is created on first run.

```yaml
diff_position: right  # right or bottom
scan_depth: 1
theme:
  cursor_bg: "237"
  border_focused: "12"
  border_normal: "8"
  title: "14"
  status_bar: "8"
  repo_name: "12"
  branch_name: "13"
  file_count: "7"
  folder_icon: "7"
  dir_name: "7"
  status_staged: "10"
  status_added: "10"
  status_deleted: "9"
  status_modified: "11"
  status_untracked: "8"
  default_icon: "7"
```

Theme values accept ANSI color codes (`0`-`255`) or hex colors (`"#FF79C6"`).

## Features

- Scans for git repos automatically (current directory + two levels deep)
- File watcher auto-refreshes when files change on disk
- Colored inline diffs with staged/unstaged detection
- Nerd Font file icons
- Collapsible directory tree
- Discard changes with confirmation menu
- Fully configurable color theme
