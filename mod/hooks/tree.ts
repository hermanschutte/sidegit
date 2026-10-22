import type { FileStatus, Repo } from '../types'

export type Row =
  | { kind: 'repo'; id: string; repo: Repo; collapsed: boolean }
  | { kind: 'dir'; id: string; name: string; guides: string; collapsed: boolean }
  | { kind: 'file'; id: string; file: FileStatus; guides: string }

type Dir = { name: string; path: string; dirs: Map<string, Dir>; files: FileStatus[] }

// The id of a file row is the file's absolute path.
const repoId = (repo: Repo) => `repo:${repo.path}`
const dirId = (repo: Repo, dir: string) => `dir:${repo.path}/${dir}`

// buildRows lays out the repos as sidegit does: in a repo its folders and then its root
// files, in a folder its files and then its subfolders. A collapsed row hides its children.
export function buildRows(repos: Repo[], collapsed: ReadonlySet<string>): Row[] {
  const rows: Row[] = []
  for (const repo of repos) {
    const id = repoId(repo)
    rows.push({ kind: 'repo', id, repo, collapsed: collapsed.has(id) })
    if (collapsed.has(id)) continue
    const top = folderTree(repo.files)
    addChildren(repo, [...sortedDirs(top), ...top.files], '', rows, collapsed)
  }
  return rows
}

function addChildren(repo: Repo, children: (Dir | FileStatus)[], prefix: string, rows: Row[], collapsed: ReadonlySet<string>): void {
  children.forEach((child, i) => {
    const last = i === children.length - 1
    const guides = prefix + (last ? '└ ' : '├ ')
    if (!('dirs' in child)) {
      rows.push({ kind: 'file', id: `${repo.path}/${child.path.replace(/\/$/, '')}`, file: child, guides })
      return
    }
    const id = dirId(repo, child.path)
    rows.push({ kind: 'dir', id, name: child.name, guides, collapsed: collapsed.has(id) })
    if (!collapsed.has(id)) addChildren(repo, [...child.files, ...sortedDirs(child)], prefix + (last ? '  ' : '│ '), rows, collapsed)
  })
}

function folderTree(files: FileStatus[]): Dir {
  const top: Dir = { name: '', path: '', dirs: new Map(), files: [] }
  for (const f of files) {
    // An untracked folder (a nested repo) ends in "/" and shows as one row
    const parts = f.path.replace(/\/$/, '').split('/')
    let dir = top
    for (const name of parts.slice(0, -1)) {
      let sub = dir.dirs.get(name)
      if (!sub) {
        sub = { name, path: dir.path ? `${dir.path}/${name}` : name, dirs: new Map(), files: [] }
        dir.dirs.set(name, sub)
      }
      dir = sub
    }
    dir.files.push(f)
  }
  return top
}

function sortedDirs(dir: Dir): Dir[] {
  return [...dir.dirs.values()].sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0))
}

// ancestorIds returns the ids of the repo and folder rows that hold the file at path.
export function ancestorIds(repos: Repo[], path: string): string[] {
  const repo = repos.find(r => path.startsWith(`${r.path}/`))
  if (!repo) return []
  const parts = path.slice(repo.path.length + 1).split('/').slice(0, -1)
  return [repoId(repo), ...parts.map((_, i) => dirId(repo, parts.slice(0, i + 1).join('/')))]
}

// fitRepo shortens the name and branch of a repo row to fit width, as sidegit does.
// An empty branch means the row only has room for the name.
export function fitRepo(repo: Repo, width: number): { name: string; branch: string; count: string } {
  const avail = width - 2 // the arrow and a space
  const branch = `[${repo.branch}]`
  const count = `(${repo.files.length})`
  const ab = (repo.ahead > 0 ? ` ↑${repo.ahead}` : '') + (repo.behind > 0 ? ` ↓${repo.behind}` : '')
  if (repo.name.length + 1 + branch.length + 1 + count.length + ab.length <= avail) return { name: repo.name, branch, count }
  let room = avail - count.length - ab.length - 2
  let shownCount = count
  if (room < 7) {
    room = avail - ab.length - 1
    shownCount = ''
  }
  const [name, shownBranch] = fitNameAndBranch(repo.name, branch, room)
  if (name && shownBranch) return { name, branch: shownBranch, count: shownCount }
  return { name: truncatePath(repo.name, Math.max(1, avail)), branch: '', count: '' }
}

// fitNameAndBranch gives about 60% of the room to the name and 40% to the branch,
// and gives the room one of them does not need to the other.
function fitNameAndBranch(name: string, branch: string, avail: number): [string, string] {
  if (name.length + branch.length <= avail) return [name, branch]
  const minName = 3
  const minBranch = 4
  if (avail < minName + minBranch) return [truncatePath(name, Math.max(1, avail)), '']
  let nameSpace = Math.max(minName, Math.floor((avail * 3) / 5))
  let branchSpace = avail - nameSpace
  if (branchSpace < minBranch) {
    branchSpace = minBranch
    nameSpace = avail - branchSpace
  }
  if (name.length <= nameSpace && branch.length > branchSpace) {
    branchSpace = avail - name.length
    nameSpace = name.length
  } else if (branch.length <= branchSpace && name.length > nameSpace) {
    nameSpace = avail - branch.length
    branchSpace = branch.length
  }
  return [truncatePath(name, nameSpace), truncateBranch(branch, branchSpace)]
}

// truncatePath shortens a path from the left to fit width, as "…/Projects/gitbar".
export function truncatePath(path: string, width: number): string {
  if (path.length <= width) return path
  if (width <= 3) return '...'
  const parts = path.split('/')
  let result = parts[parts.length - 1] ?? ''
  if (result.length + 2 > width) return '…' + result.slice(result.length - (width - 1))
  for (let i = parts.length - 2; i >= 0; i--) {
    const candidate = `${parts[i]}/${result}`
    if (candidate.length + 2 > width) break // 2 for the "…/" prefix
    result = candidate
  }
  return result === path ? path : `…/${result}`
}

// truncateBranch shortens "[branch]" and keeps the brackets.
function truncateBranch(branch: string, width: number): string {
  if (branch.length <= width) return branch
  if (width <= 2) return ''
  if (width === 3) return '[…]'
  return `[${branch.slice(1, 1 + width - 3)}…]`
}

// age shows how long ago a file changed: "12s", "5m", "3h" or "2d".
export function age(ms: number): string {
  const s = Math.max(0, Math.floor(ms / 1000))
  const text = s < 60 ? `${s}s` : s < 3600 ? `${Math.floor(s / 60)}m` : s < 86400 ? `${Math.floor(s / 3600)}h` : `${Math.floor(s / 86400)}d`
  return text.padStart(3)
}
