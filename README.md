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
| `d` | Discard changes, or remove a worktree (opens confirmation menu) |
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
  worktree: "6"
```

Theme values accept ANSI color codes (`0`-`255`) or hex colors (`"#FF79C6"`).

## Claude Code hook

To show which Claude Code session edited each file, add this hook to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write|MultiEdit|NotebookEdit",
        "hooks": [{ "type": "command", "command": "sidegit claude-hook" }]
      }
    ]
  }
}
```

The hook writes the session name for each edited file to the `sidegit` folder in your user cache folder. It does not record changes that Claude makes with Bash (for example `sed`). A file that changes again after the edit, for example by hand, loses its session name.

## Claude Code mod

The `mod` folder holds a version of sidegit that runs inside Claude Code, in a pane beside the chat. It shows the same tree of repos and changed files for the session's project folder. It marks the file that Claude reads (`◉`) or edits (`✎`), opens collapsed rows to show a file that Claude edits, and refreshes after each edit and each Bash command. It does not show pull requests, worktrees, session names or diffs. Use the `sidegit` binary for those.

Load it in a session:

```
claude --plugin-dir /path/to/sidegit/mod
```

The pane needs the fullscreen layout (`/tui fullscreen`) and a terminal at least 110 columns wide. `/sidegit` opens it. Tested on Claude Code 2.1.308.

The pane opens 56 columns wide. Change that with "Pane width" under sidegit in `/config`. Drag the edge of the pane to change the width for the session.

Run the tests with `claude plugin test mod`.

## Features

- Scans for git repos automatically (current directory + two levels deep)
- Shows linked git worktrees as child rows under their main repo
- File watcher auto-refreshes when files change on disk
- Colored inline diffs with staged/unstaged detection
- Lines added and deleted per file (`+12 -3`), compared to HEAD
- Time since each file changed (`5m`). A file that changed in the last 30 seconds shows in the title color
- The Claude Code session that last edited each file, by the name in its terminal tab (needs the [hook](#claude-code-hook))
- Pull request of the current branch on each repo row: number colored by state, check result (✓ ✗ ●) and unresolved review threads. Needs the [GitHub CLI](https://cli.github.com) (`gh`), logged in. Refreshes every 2 minutes and on `r`
- A macOS notification through iTerm2 when the checks of a PR finish or new review threads come in. With the repo open in several panes, only one pane sends it
- Nerd Font file icons
- Collapsible directory tree
- Discard changes with confirmation menu
- Fully configurable color theme
