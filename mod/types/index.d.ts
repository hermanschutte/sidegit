export type StatusCode = 'M' | 'A' | 'D' | 'R' | 'C' | '?'

export type FileStatus = {
  /** relative to the repo */
  path: string
  status: StatusCode
  staged: boolean
  /** lines added compared to HEAD (all lines for an untracked file) */
  added: number
  /** lines deleted compared to HEAD */
  deleted: number
  /** 0 for a deleted file */
  mtimeMs: number
}

export type Repo = {
  /** absolute */
  path: string
  /** relative to the scan root; the absolute path for the root repo; the folder name for a worktree */
  name: string
  branch: string
  ahead: number
  behind: number
  files: FileStatus[]
  /** true for a linked worktree, which follows its main repo in the list */
  worktree: boolean
}

/** A file Claude read or wrote. */
export type Touch = {
  id: string
  /** absolute */
  path: string
  kind: 'read' | 'write'
  running: boolean
  at: number
}

/** The newest GitHub pull request of a branch. */
export type PRStatus = {
  number: number
  /** OPEN, MERGED or CLOSED */
  state: string
  draft: boolean
  /** SUCCESS, FAILURE, ERROR, PENDING or EXPECTED; empty without checks */
  checks: string
  /** review threads not resolved yet */
  unresolved: number
}

declare module 'claude-code' {
  interface PluginState {
    sidegit: {
      root: string
      repos: Repo[]
      /** ids of the collapsed repo and folder rows */
      collapsed: string[]
      touches: Touch[]
      /** by repo path and branch; null when the branch has no PR */
      prs: Record<string, PRStatus | null>
    }
  }
}
