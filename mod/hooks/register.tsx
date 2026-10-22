import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { FileStatus, Repo, Touch } from '../types'
import { findRepos, readRepo } from './git'
import { age, ancestorIds, buildRows, fitRepo } from './tree'

const PANE = 'sidegit'
const POLL_MS = 10_000 // the sidegit default poll_interval
const RECENT_MS = 30_000 // a file that changed less than this ago shows in the title color
const TOUCH_MS = 8_000 // how long a file Claude read or wrote stays marked after the call
const MAX_TOUCHES = 20

// The theme in ~/.config/sidegit/config.yaml
const COLOR = {
  title: '#BD93F9',
  statusBar: '#6272A4',
  repoName: '#FF79C6',
  branchName: '#8BE9FD',
  fileCount: '#6272A4',
  dirName: '#6272A4',
  staged: '#50FA7B',
  added: '#50FA7B',
  deleted: '#FF5555',
  modified: '#FFB86C',
  untracked: '#6272A4',
  treeLines: '#6272A4',
  read: '#8BE9FD',
  write: '#F1FA8C',
}
const MARK = { read: '◉', write: '✎' }
// The pulse on a marked file name: from the mark color to the white center of the band
const PULSE = {
  read: ['#8BE9FD', '#ADF0FE', '#D0F7FE', '#FFFFFF'],
  write: ['#F1FA8C', '#F5FBAF', '#FAFDD2', '#FFFFFF'],
}

const root = atom({ plugin: 'sidegit', key: 'root' } as const, '')
const repos = atom({ plugin: 'sidegit', key: 'repos' } as const, [] as Repo[])
const collapsed = atom({ plugin: 'sidegit', key: 'collapsed' } as const, [] as string[])
const touches = atom({ plugin: 'sidegit', key: 'touches' } as const, [] as Touch[])

let refreshing: Promise<void> | null = null
let again = false

// refresh finds the repos under the project folder and reads the git status of each.
// A call while a scan runs makes that scan run once more.
function refresh($: EngineInterface): Promise<void> {
  if (refreshing) {
    again = true
    return refreshing
  }
  refreshing = (async () => {
    do {
      again = false
      const dir = await $.session.root()
      const paths = await findRepos($, dir).catch(() => [])
      const list = await Promise.all(paths.map(path => readRepo($, dir, path)))
      await update($, root, () => dir)
      await update($, repos, () => list)
    } while (again)
  })()
    .catch(err => $.ui.log(`sidegit: ${String(err)}`))
    .finally(() => {
      refreshing = null
    })
  return refreshing
}

// afterChange shows a change that Claude made. A file Claude wrote is revealed in the tree.
async function afterChange($: EngineInterface, path: string): Promise<void> {
  await refresh($)
  if (!path) return
  const reveal = new Set(ancestorIds(await read($, repos), path))
  await update($, collapsed, ids => ids.filter(id => !reveal.has(id)))
  await $.ui.scroll({ to: { key: `row:${path}` }, in: PANE }).catch(() => undefined)
}

function statusColor(f: FileStatus): string | undefined {
  if (f.staged) return COLOR.staged
  if (f.status === 'A') return COLOR.added
  if (f.status === 'D') return COLOR.deleted
  if (f.status === 'M') return COLOR.modified
  if (f.status === '?') return COLOR.untracked
  return undefined
}

export const register: Register = (on, options) => {
  // The "Pane width" setting in /config. A width the person drags wins.
  const columns = typeof options?.width === 'number' ? Math.max(24, Math.round(options.width)) : 56

  on('session.start', async ($, e, next) => {
    await $.command.register({ name: 'sidegit', description: 'Show the git changes of every repo in the project folder' })
    void $.ui.open({ id: PANE, title: 'sidegit', columns })
    void refresh($)
    $.clock.every(POLL_MS, () => {
      void $.ui.panes().then(panes => (panes.some(p => p.id === PANE) ? refresh($) : undefined))
    })
    return next(e)
  })

  on('command.run', { command: 'sidegit' }, async ($, e) => {
    // A terminal docks a pane on the side only in the fullscreen layout, from 110 columns
    const terminal = await $.session.surfaces().then(
      surfaces => surfaces.includes('terminal'),
      () => true,
    )
    if (terminal && !e.presentation.isFullscreen) return { text: 'sidegit shows on the side only in the fullscreen layout. Run /tui fullscreen, then /sidegit.' }
    if (terminal && e.presentation.columns < 110) return { text: 'sidegit shows on the side only in a terminal at least 110 columns wide. Widen the terminal, then run /sidegit.' }
    const opened = await $.ui.open({ id: PANE, title: 'sidegit', columns })
    void refresh($)
    return { text: opened.isPlaced ? 'sidegit pane opened.' : `The sidegit pane is not shown: ${opened.reason}` }
  })

  on('tool.call', async ($, e, next) => {
    const kind = e.tool === 'Read' ? 'read' : e.tool === 'Edit' || e.tool === 'Write' || e.tool === 'NotebookEdit' ? 'write' : null
    const path = e.tool === 'NotebookEdit' ? e.notebook_path : e.tool === 'Read' || e.tool === 'Edit' || e.tool === 'Write' ? e.file_path : ''
    const id = e.tool_use_id
    if (kind && path) {
      const touch: Touch = { id, path, kind, running: true, at: await $.clock.now() }
      await update($, touches, list => [...list.filter(t => t.path !== path), touch].slice(-MAX_TOUCHES))
    }
    const result = await next(e)
    if (kind && path) {
      const at = await $.clock.now()
      await update($, touches, list => (result.deny ? list.filter(t => t.id !== id) : list.map(t => (t.id === id ? { ...t, running: false, at } : t))))
      $.clock.after(TOUCH_MS, () => $.ui.invalidate('ui.render'))
    }
    if (!result.deny && (kind === 'write' || e.tool === 'Bash')) void afterChange($, kind === 'write' ? path : '')
    return result
  })

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const { Box, Text, Button } = $.ui.resolve(e)
    // The terminal and the desktop app draw a Client; VS Code and the mobile app do not
    const Client = e.surface === 'terminal' || e.surface === 'desktop' ? $.ui.resolve(e).Client : undefined
    // On the terminal's main screen the pane that session start opened sits above the prompt
    if (e.surface === 'terminal' && e.props.placement === 'inline') {
      void $.ui.close({ id: PANE }).catch(() => undefined)
      return <Box />
    }
    const [dir, list, closed, touched, now] = await Promise.all([read($, root), read($, repos), read($, collapsed), read($, touches), $.clock.now()])
    const width = Math.max(20, e.props.bodyColumns)
    const live = new Map(touched.filter(t => t.running || now - t.at < TOUCH_MS).map(t => [t.path, t]))
    const latest = [...live.values()].pop()
    const changes = list.reduce((n, r) => n + r.files.length, 0)
    const toggle = (id: string) => void update($, collapsed, ids => (ids.includes(id) ? ids.filter(x => x !== id) : [...ids, id]))

    if (list.length === 0) return <Text color={COLOR.statusBar}>{dir ? `No git repositories found in ${dir}.` : 'Scanning for git repositories…'}</Text>

    return (
      <Box flexDirection="column">
        <Text color={COLOR.statusBar} wrap="truncate-end">
          {` ${list.length} repo(s) | ${changes} change(s)`}
          {latest ? <Text color={COLOR[latest.kind]}>{` | ${MARK[latest.kind]} ${latest.path.startsWith(`${dir}/`) ? latest.path.slice(dir.length + 1) : latest.path}`}</Text> : null}
        </Text>
        {buildRows(list, new Set(closed)).map(row => {
          if (row.kind === 'repo') {
            const fit = fitRepo(row.repo, width)
            return (
              <Button key={row.id} plain onPress={() => toggle(row.id)}>
                {row.collapsed ? '▸ ' : '▾ '}
                <Text bold color={COLOR.repoName}>{fit.name}</Text>
                {fit.branch ? <Text color={COLOR.branchName}>{` ${fit.branch}`}</Text> : null}
                {fit.count ? <Text color={COLOR.fileCount}>{` ${fit.count}`}</Text> : null}
                {fit.branch && row.repo.ahead > 0 ? <Text color={COLOR.added}>{` ↑${row.repo.ahead}`}</Text> : null}
                {fit.branch && row.repo.behind > 0 ? <Text color={COLOR.deleted}>{` ↓${row.repo.behind}`}</Text> : null}
              </Button>
            )
          }
          if (row.kind === 'dir') {
            return (
              <Box flexDirection="row">
                <Text color={COLOR.treeLines}>{row.guides}</Text>
                <Button key={row.id} plain onPress={() => toggle(row.id)}>
                  {row.collapsed ? '▸ ' : '▾ '}
                  <Text bold color={COLOR.dirName}>{row.name}</Text>
                </Button>
              </Box>
            )
          }
          const f = row.file
          const touch = live.get(row.id)
          const since = now - f.mtimeMs
          const recent = f.mtimeMs > 0 && since < RECENT_MS
          const name = row.id.slice(row.id.lastIndexOf('/') + 1)
          const pulse =
            Client && touch ? (
              <Client key={`pulse:${row.id}`} module="./pulse.tsx" props={{ text: ` ${MARK[touch.kind]} ${name}`, colors: PULSE[touch.kind] }} />
            ) : null
          return (
            <Box key={`row:${row.id}`} flexDirection="row">
              <Box flexShrink={1} overflow="hidden">
                <Text wrap="truncate-end">
                  <Text color={COLOR.treeLines}>{row.guides}</Text>
                  <Text bold={f.staged} color={statusColor(f)}>{f.status}</Text>
                  {pulse ? null : touch ? (
                    <Text bold color={COLOR[touch.kind]}>{` ${MARK[touch.kind]} ${name}`}</Text>
                  ) : (
                    <Text bold={recent} color={recent ? COLOR.title : undefined}>{` ${name}`}</Text>
                  )}
                </Text>
                {pulse}
              </Box>
              <Box flexGrow={1} />
              <Text>
                {f.added > 0 ? <Text color={COLOR.added}>{` +${f.added}`}</Text> : null}
                {f.deleted > 0 ? <Text color={COLOR.deleted}>{` -${f.deleted}`}</Text> : null}
                {f.mtimeMs > 0 ? <Text color={recent ? COLOR.title : COLOR.untracked}>{` ${age(since)}`}</Text> : null}
              </Text>
            </Box>
          )
        })}
      </Box>
    )
  })
}
