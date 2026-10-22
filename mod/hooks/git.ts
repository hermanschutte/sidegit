import type { EngineInterface } from 'claude-code'

import type { FileStatus, Repo, StatusCode } from '../types'

const TIMEOUT_MS = 15_000
// An untracked file over 1 MB counts as 0 lines, as in sidegit.
const MAX_COUNT_BYTES = 1 << 20

// findRepos returns the git repos in root and up to two levels below it, root first.
export async function findRepos($: EngineInterface, root: string): Promise<string[]> {
  const run = await $.process.run(['find', root, '-maxdepth', '3', '-name', '.git', '-type', 'd'], { timeoutMs: TIMEOUT_MS })
  return parseFind(run.stdout, root)
}

export function parseFind(stdout: string, root: string): string[] {
  const repos: string[] = []
  for (const line of stdout.split('\n')) {
    if (!line.endsWith('/.git')) continue
    const repo = line.slice(0, -'/.git'.length)
    const rel = repo === root ? '' : repo.slice(root.length + 1)
    // Hidden folders are skipped, as in sidegit
    if (rel.split('/').some(part => part.startsWith('.'))) continue
    repos.push(repo)
  }
  return repos.sort((a, b) => (a === root ? -1 : b === root ? 1 : a < b ? -1 : a > b ? 1 : 0))
}

export async function readRepo($: EngineInterface, root: string, path: string): Promise<Repo> {
  const [status, numstat] = await Promise.all([
    git($, path, ['status', '--porcelain=v2', '--branch', '--untracked-files=all', '-z']),
    git($, path, ['diff', 'HEAD', '--numstat', '--no-renames', '-z']),
  ])
  const repo: Repo = { path, name: path === root ? path : path.slice(root.length + 1), ...parseStatus(status) }
  const counts = parseNumstat(numstat)
  await Promise.all(
    repo.files.map(async f => {
      const abs = `${path}/${f.path}`
      try {
        const stat = await $.fs.stat(abs)
        f.mtimeMs = stat.mtimeMs
        if (f.status === '?') {
          f.added = stat.kind === 'file' && stat.size <= MAX_COUNT_BYTES ? await countLines($, abs, stat.mtimeMs) : 0
          return
        }
      } catch {
        // a deleted file
      }
      const c = counts.get(f.path)
      if (c) [f.added, f.deleted] = c
    }),
  )
  return repo
}

// git returns the command's output, or '' when it fails (a repo without commits has no HEAD to diff).
async function git($: EngineInterface, repo: string, args: string[]): Promise<string> {
  try {
    const run = await $.process.run(['git', '--no-optional-locks', '-C', repo, ...args], { timeoutMs: TIMEOUT_MS })
    return run.exitCode === 0 ? run.stdout : ''
  } catch {
    return ''
  }
}

export function parseStatus(out: string): Pick<Repo, 'branch' | 'ahead' | 'behind' | 'files'> {
  const result = { branch: 'unknown', ahead: 0, behind: 0, files: [] as FileStatus[] }
  const records = out.split('\0')
  for (let i = 0; i < records.length; i++) {
    const rec = records[i] ?? ''
    if (rec.startsWith('# branch.head ')) {
      const head = rec.slice('# branch.head '.length)
      result.branch = head === '(detached)' ? 'HEAD' : head
    } else if (rec.startsWith('# branch.ab ')) {
      const m = /^# branch\.ab \+(\d+) -(\d+)$/.exec(rec)
      if (m) [result.ahead, result.behind] = [Number(m[1]), Number(m[2])]
    } else if (rec.startsWith('1 ') || rec.startsWith('2 ')) {
      // 1 XY sub mH mI mW hH hI path
      // 2 XY sub mH mI mW hH hI Xscore path, then the original path as the next record
      const fields = rec.split(' ')
      const path = fields.slice(rec[0] === '1' ? 8 : 9).join(' ')
      if (rec[0] === '2') i++
      const file = fromXY(fields[1] ?? '..', path)
      if (file) result.files.push(file)
    } else if (rec.startsWith('? ')) {
      result.files.push(newFile(rec.slice(2), '?', false))
    }
  }
  return result
}

// fromXY prefers the unstaged change; with none it shows the staged one.
function fromXY(xy: string, path: string): FileStatus | null {
  const staged = xy[0] ?? '.'
  const unstaged = xy[1] ?? '.'
  if (unstaged !== '.') return newFile(path, statusCode(unstaged), false)
  if (staged !== '.') return newFile(path, statusCode(staged), true)
  return null
}

function statusCode(c: string): StatusCode {
  return c === 'A' || c === 'D' || c === 'R' || c === 'C' ? c : 'M'
}

function newFile(path: string, status: StatusCode, staged: boolean): FileStatus {
  return { path, status, staged, added: 0, deleted: 0, mtimeMs: 0 }
}

// parseNumstat returns added and deleted lines per tracked file. Binary files get no entry.
export function parseNumstat(out: string): Map<string, [number, number]> {
  const counts = new Map<string, [number, number]>()
  for (const rec of out.split('\0')) {
    const m = /^(\d+)\t(\d+)\t([\s\S]*)$/.exec(rec)
    if (m) counts.set(m[3] ?? '', [Number(m[1]), Number(m[2])])
  }
  return counts
}

const lineCache = new Map<string, { mtimeMs: number; lines: number }>()

// countLines counts the lines of an untracked text file. A binary file counts as 0.
async function countLines($: EngineInterface, abs: string, mtimeMs: number): Promise<number> {
  const hit = lineCache.get(abs)
  if (hit?.mtimeMs === mtimeMs) return hit.lines
  let lines = 0
  try {
    const text = await $.fs.read(abs)
    if (typeof text === 'string' && !text.includes('\0')) lines = text.split('\n').length - (text === '' || text.endsWith('\n') ? 1 : 0)
  } catch {
    // unreadable counts as 0
  }
  lineCache.set(abs, { mtimeMs, lines })
  return lines
}
