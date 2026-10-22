// Nerd Font icons, as in the sidegit binary (tree.go), codepoints from nvim-web-devicons.

export const FOLDER_ICON = ''
const DEFAULT_ICON = ''
// The theme key for an icon with no color of its own: the text color, as the binary's
// default_icon is the Dracula foreground
const DEFAULT_COLOR = 'text'

const BY_EXT: Record<string, string> = {
  '.go': '', // seti-go
  '.js': '', // seti-javascript
  '.ts': '', // seti-typescript
  '.tsx': '', // seti-react
  '.jsx': '', // seti-react
  '.py': '', // seti-python
  '.rb': '', // seti-ruby
  '.rs': '', // seti-rust
  '.java': '', // dev-java
  '.c': '', // seti-c
  '.cpp': '', // seti-cpp
  '.h': '', // seti-c
  '.cs': '\u{F0911}', // md-language_csharp
  '.php': '', // seti-php
  '.swift': '', // dev-swift
  '.kt': '', // seti-kotlin
  '.html': '', // dev-html5
  '.css': '', // seti-css
  '.scss': '', // seti-sass
  '.json': '', // seti-json
  '.yaml': '', // seti-yaml
  '.yml': '', // seti-yaml
  '.toml': '', // seti-toml
  '.xml': '\u{F05C0}', // md-xml
  '.md': '', // oct-markdown
  '.sh': '', // seti-shell
  '.bash': '', // seti-shell
  '.zsh': '', // seti-shell
  '.fish': '', // seti-shell
  '.sql': '', // dev-database
  '.svg': '\u{F0721}', // md-svg
  '.png': '', // seti-image
  '.jpg': '', // seti-image
  '.jpeg': '', // seti-image
  '.gif': '', // seti-image
  '.vue': '', // seti-vue
  '.svelte': '', // seti-svelte
  '.lua': '', // seti-lua
  '.vim': '', // seti-vim
  '.lock': '', // seti-lock
  '.env': '', // oct-key
  '.gitignore': '', // dev-git
  '.mod': '', // seti-go
  '.sum': '', // seti-go
}

// Whole file names, matched before the extension
const BY_NAME: Record<string, string> = {
  Dockerfile: '\u{F01A8}', // md-docker
  Makefile: '', // seti-makefile
  LICENSE: '', // seti-license
  '.gitignore': '', // dev-git
  'go.mod': '', // seti-go
  'go.sum': '', // seti-go
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
