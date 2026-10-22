// Nerd Font icons, as in the sidegit binary (tree.go), codepoints from nvim-web-devicons.

export const FOLDER_ICON = '\uf07b'
export const WORKTREE_ICON = '\ue725'
// The icon before the count of unresolved review threads on a PR
export const COMMENT_ICON = '\uf075'
const DEFAULT_ICON = '\uf15b'
// The theme key for an icon with no color of its own: the text color, as the binary's
// default_icon is the Dracula foreground
const DEFAULT_COLOR = 'text'

const BY_EXT: Record<string, string> = {
  '.go': '\ue627', // seti-go
  '.js': '\ue60c', // seti-javascript
  '.ts': '\ue628', // seti-typescript
  '.tsx': '\ue7ba', // seti-react
  '.jsx': '\ue625', // seti-react
  '.py': '\ue606', // seti-python
  '.rb': '\ue791', // seti-ruby
  '.rs': '\ue68b', // seti-rust
  '.java': '\ue738', // dev-java
  '.c': '\ue61e', // seti-c
  '.cpp': '\ue61d', // seti-cpp
  '.h': '\ue61e', // seti-c
  '.cs': '\u{F0911}', // md-language_csharp
  '.php': '\ue608', // seti-php
  '.swift': '\ue755', // dev-swift
  '.kt': '\ue634', // seti-kotlin
  '.html': '\ue736', // dev-html5
  '.css': '\ue6b8', // seti-css
  '.scss': '\ue603', // seti-sass
  '.json': '\ue60b', // seti-json
  '.yaml': '\ue615', // seti-yaml
  '.yml': '\ue615', // seti-yaml
  '.toml': '\ue6b2', // seti-toml
  '.xml': '\u{F05C0}', // md-xml
  '.md': '\uf48a', // oct-markdown
  '.sh': '\ue795', // seti-shell
  '.bash': '\ue795', // seti-shell
  '.zsh': '\ue795', // seti-shell
  '.fish': '\ue795', // seti-shell
  '.sql': '\ue706', // dev-database
  '.svg': '\u{F0721}', // md-svg
  '.png': '\ue60d', // seti-image
  '.jpg': '\ue60d', // seti-image
  '.jpeg': '\ue60d', // seti-image
  '.gif': '\ue60d', // seti-image
  '.vue': '\ue6a0', // seti-vue
  '.svelte': '\ue697', // seti-svelte
  '.lua': '\ue620', // seti-lua
  '.vim': '\ue62b', // seti-vim
  '.lock': '\ue672', // seti-lock
  '.env': '\uf462', // oct-key
  '.gitignore': '\ue702', // dev-git
  '.mod': '\ue627', // seti-go
  '.sum': '\ue627', // seti-go
}

// Whole file names, matched before the extension
const BY_NAME: Record<string, string> = {
  Dockerfile: '\u{F01A8}', // md-docker
  Makefile: '\ue673', // seti-makefile
  LICENSE: '\ue62f', // seti-license
  '.gitignore': '\ue702', // dev-git
  'go.mod': '\ue627', // seti-go
  'go.sum': '\ue627', // seti-go
}

// File-type colors, which no Claude Code theme has
const COLORS: Record<string, string> = {
  '.go': '#00ADD8', '.js': '#CBCB41', '.ts': '#519ABA', '.tsx': '#1354BF',
  '.jsx': '#20C2E3', '.py': '#FFBC03', '.rb': '#701516', '.rs': '#DEA584',
  '.java': '#CC3E44', '.html': '#E44D26', '.css': '#663399', '.scss': '#F55385',
  '.json': '#CBCB41', '.yaml': '#6D8086', '.yml': '#6D8086', '.toml': '#9C4221',
  '.md': '#DDDDDD', '.sh': '#4D5A5E', '.bash': '#4D5A5E', '.zsh': '#4D5A5E',
  '.php': '#A074C4', '.swift': '#E37933', '.kt': '#7F52FF', '.lua': '#51A0CF',
  '.vue': '#8DC149', '.svelte': '#FF3E00', '.sql': '#DAD8D8', '.lock': '#BBBBBB',
  '.env': '#FAF743', '.gitignore': '#F54D27',
}

// fileIcon returns the icon of a file name and its color: a hex color, or a theme key.
export function fileIcon(name: string): { icon: string; color: string } {
  const ext = name.includes('.') ? name.slice(name.lastIndexOf('.')).toLowerCase() : ''
  const icon = BY_NAME[name] ?? BY_EXT[ext]
  if (!icon) return { icon: DEFAULT_ICON, color: DEFAULT_COLOR }
  return { icon, color: COLORS[ext] ?? DEFAULT_COLOR }
}
